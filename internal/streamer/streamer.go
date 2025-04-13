package streamer

import (
	"context"
	"fmt"
	"net"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/lib/log"
	"time"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/nestor-poliakov/joy5/format/flv/flvio"
	"github.com/nestor-poliakov/joy5/format/rtmp"
)

type Streamer struct {
	playlistLogic *logic.Playlist
	url           string
	client        *rtmp.Client
	conn          *rtmp.Conn
	nconn         net.Conn
	rl            *rateLimiter
	configs       []av.Packet
	ch            <-chan piece
	curVideoId    int64
}

func NewStreamer(ch <-chan piece, stream app.Stream, playlistLogic *logic.Playlist) *Streamer {
	return &Streamer{
		playlistLogic: playlistLogic,
		url:           stream.Settings.Url,
		client:        rtmp.NewClient(),
		ch:            ch,
		rl:            newRateLimiter(),
		configs:       make([]av.Packet, 0, 3),
	}
}

func (s *Streamer) reconnect() (err error) {
	if s.nconn != nil {
		s.nconn.Close()
	}
	if s.url == "" {
		return fmt.Errorf("empty streaming url")
	}
	s.conn, s.nconn, err = s.client.Dial(s.url, rtmp.PrepareWriting)
	if err != nil {
		return fmt.Errorf("failed to connect to rtmp server %q: %w", s.url, err)
	}
	for i := range s.configs {
		err = s.conn.WritePacket(s.configs[i])
		if err != nil {
			return fmt.Errorf("write %s config: %w", av.PacketTypeString[s.configs[i].Type], err)
		}
	}
	return nil
}

func (s *Streamer) Run(ctx context.Context, wg *sync.WaitGroup) error {
	ctx = log.With(ctx, "worker", "streamer")
	err := s.reconnect()
	if err != nil {
		return err
	}
	wg.Add(1)
	go s.streamingLoop(ctx, wg)
	return nil
}

func (s *Streamer) streamingLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer log.FromContext(ctx).Info("streaming ended; closing connection")
	defer s.nconn.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case piece, ok := <-s.ch:
			if !ok {
				return
			}
			vctx := log.With(ctx, "video_id", piece.videoId)
			err := s.processPackets(vctx, piece)
			if err != nil {
				log.FromContexts(vctx).With("error", err).Errorf("process %d packets", len(piece.packets))
				continue
			}
		}
	}
}

func (s *Streamer) processPackets(ctx context.Context, piece piece) error {
	if piece.videoId != s.curVideoId {
		s.playlistLogic.SetCurrent(piece.videoId)
		s.rl = newRateLimiter()
		s.configs = s.configs[:0]
		s.curVideoId = piece.videoId
		log.FromContext(ctx).Info("start streaming new video")
	}
	if len(piece.packets) == 0 {
		return nil
	}
	config, videos, audios := calcPackets(piece.packets)
	log.FromContexts(ctx).Debugf("processing %d packets from %s to %s; c: %d a: %d v: %d", len(piece.packets), piece.packets[0].Time, piece.packets[len(piece.packets)-1].Time, config, audios, videos)
	reconnects := 0
	for i := range piece.packets {
		err := s.processPacket(piece.packets[i])
		if err != nil {
			if reconnects > 10 {
				var t *time.Timer
				if reconnects > 20 {
					t = time.NewTimer(time.Minute)
				} else {
					t = time.NewTimer(time.Second * time.Duration(reconnects))
				}
				select {
				case <-t.C:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			log.FromContexts(ctx).With("error", err).Error("processing packet %q; reconnecting", av.PacketTypeString[piece.packets[i].Type])
			s.reconnect()
			reconnects++
			i--
		}
	}
	return nil
}

func (s *Streamer) processPacket(packet av.Packet) error {
	if packet.Type == av.Metadata {
		arr, _ := flvio.ParseAMFVals(packet.Data, false)
		m := arr[0].(flvio.AMFMap)
		m.Set("duration", 0)
		m.Set("filesize", 0)
		data := make([]byte, flvio.FillAMF0Vals(nil, []any{m}))
		flvio.FillAMF0Vals(data, []any{m})
		packet.Data = data
	}
	if packet.Type <= 2 {
		s.rl.Limit(packet)
	} else {

		s.configs = append(s.configs, packet)
	}
	return s.conn.WritePacket(packet)
}

func calcPackets(packets []av.Packet) (configs int, video int, audio int) {
	for _, p := range packets {
		switch p.Type {
		case av.AAC:
			audio++
		case av.H264:
			video++
		default:
			configs++
		}
	}
	return
}
