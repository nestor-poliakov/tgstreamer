package streamer

import (
	"context"
	"fmt"
	"net"
	"tgstreamer/lib/log"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/nestor-poliakov/joy5/format/flv/flvio"
	"github.com/nestor-poliakov/joy5/format/rtmp"
)

type Streamer struct {
	url        string
	client     *rtmp.Client
	conn       *rtmp.Conn
	nconn      net.Conn
	rl         *rateLimiter
	configs    []av.Packet
	ch         <-chan piece
	curVideoId int64
}

func NewStreamer(streamingUrl string, ch <-chan piece) *Streamer {
	client := rtmp.NewClient()

	return &Streamer{
		url:     streamingUrl,
		client:  client,
		ch:      ch,
		rl:      newRateLimiter(),
		configs: make([]av.Packet, 0, 3),
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

	return nil
}

func (s *Streamer) Run(ctx context.Context) error {
	ctx = log.With(ctx, "worker", "streamer")
	err := s.reconnect()
	if err != nil {
		return err
	}
	go s.streamingLoop(ctx)
	return nil
}

func (s *Streamer) streamingLoop(ctx context.Context) {
	defer func() {
		log.FromContext(ctx).Info("streaming ended; closing connection")
		s.nconn.Close()
	}()
	videoCtx := ctx
	for {
		select {
		case <-ctx.Done():
			return
		case piece, ok := <-s.ch:
			if !ok {
				return
			}
			if piece.videoId != s.curVideoId {
				s.rl = newRateLimiter()
				s.configs = s.configs[:0]
				s.curVideoId = piece.videoId
				videoCtx = log.With(ctx, "video_id", piece.videoId)
				log.FromContext(videoCtx).Info("start streaming new video")
			}
			if len(piece.packets) == 0 {
				continue
			}
			err := s.processPackets(videoCtx, piece.packets)
			if err != nil {
				log.FromContexts(videoCtx).With("error", err).Errorf("process %d packets", len(piece.packets))
				continue
			}
		}
	}
}

func (s *Streamer) processPackets(ctx context.Context, packets []av.Packet) error {
	config, videos, audios := calcPackets(packets)
	log.FromContexts(ctx).Debugf("processing %d packets from %s to %s; c: %d a: %d v: %d", len(packets), packets[0].Time, packets[len(packets)-1].Time, config, audios, videos)
	for i := range packets {
		err := s.processPacket(packets[i])
		if err != nil {
			log.FromContexts(ctx).With("error", err).Error("processing packet %q; reconnecting", av.PacketTypeString[packets[i].Type])
			s.reconnect()
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
