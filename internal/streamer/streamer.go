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
	"github.com/nestor-poliakov/joy5/format/rtmp"
)

type Streamer struct {
	playlistLogic     *logic.Playlist
	play              *logic.Play
	stream            app.Stream
	client            *rtmp.Client
	conn              *rtmp.Conn
	nconn             net.Conn
	rl                *rateLimiter
	configs           []av.Packet
	ch                <-chan gop
	toSkip            chan int64
	curVideoId        int64
	curPlaylistItemId int64
	skip              bool
}

func NewStreamer(ch <-chan gop, stream app.Stream, playlistLogic *logic.Playlist, play *logic.Play) *Streamer {
	return &Streamer{
		playlistLogic: playlistLogic,
		play:          play,
		stream:        stream,
		client:        rtmp.NewClient(),
		ch:            ch,
		toSkip:        make(chan int64, 1),
		rl:            newRateLimiter(),
		configs:       make([]av.Packet, 0, 3),
	}
}

func (s *Streamer) connect() (err error) {
	s.closeConn()
	if s.stream.Settings.Url == "" {
		return fmt.Errorf("empty streaming url")
	}
	s.conn, s.nconn, err = s.client.Dial(s.stream.Settings.Url, rtmp.PrepareWriting)
	if err != nil {
		return fmt.Errorf("failed to connect to rtmp server %q: %w", s.stream.Settings.Url, err)
	}
	return nil
}

func (s *Streamer) closeConn() {
	if s.nconn != nil {
		s.nconn.Close()
	}
}

func (s *Streamer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx = log.With(ctx, "worker", "streamer")
	s.reconnect(ctx)
	s.skip = false
	wg.Add(1)
	go s.streamingLoop(ctx, wg)
}

func (s *Streamer) Skip(playlistItemId int64) {
	s.toSkip <- playlistItemId
}

func (s *Streamer) streamingLoop(ctx context.Context, wg *sync.WaitGroup) {
	log.FromContext(ctx).Info("streamer started")
	defer wg.Done()
	defer log.FromContext(ctx).Info("streaming ended; closing connection")
	defer s.closeConn()
	for {
		select {
		case <-ctx.Done():
			return
		case playlistItemId := <-s.toSkip:
			if s.curPlaylistItemId == playlistItemId {
				s.skip = true
			}
		case g, ok := <-s.ch:
			if !ok {
				return
			}
			vctx := log.With(ctx, "video_id", g.video.Id, "playlist_item_id", g.playlistItemId)
			err := s.processPackets(vctx, g)
			if err != nil {
				log.FromContexts(vctx).With("error", err).Errorf("process %d packets", len(g.packets))
			}
		}
	}
}

func (s *Streamer) processPackets(ctx context.Context, g gop) error {
	// Check if this is a new video or a restart of the same video
	isNewVideoInstance := g.video.Id != s.curVideoId ||
		(len(g.packets) > 0 && g.packets[0].Type == av.Metadata)

	if isNewVideoInstance {
		s.playlistLogic.SetCurrent(g.playlistItemId)
		s.play.Announce(g.playlistItemId)
		s.rl = newRateLimiter()
		s.configs = s.configs[:0]
		s.curVideoId = g.video.Id
		s.curPlaylistItemId = g.playlistItemId
		s.skip = false
		log.FromContexts(ctx).Infof("start streaming new video %d", g.video.Id)
	}
	if len(g.packets) == 0 {
		return nil
	}
	if s.skip {
		log.FromContexts(ctx).Infof("skipping %d packets", len(g.packets))
		return nil
	}
	config, videos, audios := calcPackets(g.packets)
	log.FromContexts(ctx).Debugf("processing %d packets from %s to %s; c: %d a: %d v: %d", len(g.packets), g.packets[0].Time, g.packets[len(g.packets)-1].Time, config, audios, videos)
	for i := range g.packets {
		err := s.processPacket(g.packets[i])
		if err == nil {
			continue
		}
		log.FromContexts(ctx).With("error", err).Errorf("processing packet %q; reconnecting", av.PacketTypeString[g.packets[i].Type])
		s.reconnect(ctx)
	}
	return nil
}

func (s *Streamer) reconnect(ctx context.Context) {
	reconnects := 0
	for {
		var t *time.Timer
		if reconnects > 20 {
			t = time.NewTimer(time.Minute)
		} else {
			t = time.NewTimer(time.Second * time.Duration(reconnects))
		}
		select {
		case <-t.C:
			t.Stop()
		case <-ctx.Done():
			return
		}

		err := s.connect()
		if err != nil {
			log.FromContexts(ctx).With("error", err).Errorf("attempt %d reconnection error", reconnects)
			reconnects++
		} else {
			log.FromContexts(ctx).Infof("attempt %d reconnection success; skip packages until new video", reconnects)
			s.skip = true
			return
		}
	}
}

func (s *Streamer) processPacket(packet av.Packet) error {
	isMedia := packet.Type <= 2
	if isMedia {
		s.rl.Limit(packet)
	} else {
		s.configs = append(s.configs, packet)
	}
	err := s.conn.WritePacket(packet)
	if isMedia {
		s.rl.Mark()
	}
	return err
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
