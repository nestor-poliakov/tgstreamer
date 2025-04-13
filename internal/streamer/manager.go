package streamer

import (
	"context"
	"fmt"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/postgres"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
	"time"
)

type Manager struct {
	streams       map[int64]*stream
	downloader    *rpc.YtDlpClient
	playlistLogic *logic.Playlist
	videoLogic    *logic.Video
	streamStorage postgres.Stream
	wg            *sync.WaitGroup
	stopFunc      func()
}

func NewManager(playlistLogic *logic.Playlist, videoLogic *logic.Video, streamStorage postgres.Stream, downloader *rpc.YtDlpClient) *Manager {
	m := &Manager{
		streams:       make(map[int64]*stream),
		playlistLogic: playlistLogic,
		videoLogic:    videoLogic,
		streamStorage: streamStorage,
		downloader:    downloader,
		wg:            &sync.WaitGroup{},
		stopFunc:      func() {},
	}
	return m
}

func (m *Manager) Run(ctx context.Context) {
	ctx, m.stopFunc = context.WithCancel(ctx)
	m.wg.Add(1)
	go m.processingLoop(ctx)
}

func (m *Manager) Stop() {
	m.stopFunc()
	m.wg.Wait()
}

func (m *Manager) processingLoop(ctx context.Context) {
	defer m.wg.Done()
	err := m.updateStreams(ctx)
	if err != nil {
		log.FromContext(ctx).Error("update streams", "error", err)
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := m.updateStreams(ctx)
			if err != nil {
				log.FromContext(ctx).Error("update streams", "error", err)
			}
		case <-ctx.Done():
			for _, v := range m.streams {
				v.Stop()
			}
			return
		}
	}
}

func (m *Manager) updateStreams(ctx context.Context) error {
	log.FromContext(ctx)
	streams, err := m.streamStorage.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get all streams: %w", err)
	}
	streamsMap := map[int64]bool{}
	for _, stream := range streams {
		streamsMap[stream.Id] = true
		_, ok := m.streams[stream.Id]
		if !ok {
			m.runStream(ctx, stream)
		}
	}
	for id := range m.streams {
		if !streamsMap[id] {
			m.stopStream(ctx, id)
		}
	}
	return nil
}

func (m *Manager) runStream(ctx context.Context, strm app.Stream) {
	toDownloader := make(chan app.Video, 0)
	toReader := make(chan app.Video, 0)
	toAdCutter := make(chan piece, 0)
	toStreamer := make(chan piece, 20)
	s := &stream{
		stream:     strm,
		playlist:   NewPlaylist(toDownloader, strm, m.playlistLogic),
		downloader: NewDownloader(toDownloader, toReader, m.downloader, m.videoLogic),
		reader:     NewReader(toReader, toAdCutter),
		adCutter:   NewAdCutter(toAdCutter, toStreamer),
		streamer:   NewStreamer(toStreamer, strm, m.playlistLogic),
		wg:         &sync.WaitGroup{},
		stopFunc:   func() {},
	}
	s.Run(ctx)
	m.streams[s.stream.Id] = s
}

func (m *Manager) stopStream(ctx context.Context, id int64) {
	s, ok := m.streams[id]
	if ok {
		s.Stop()
		delete(m.streams, id)
	}
}

type stream struct {
	stream     app.Stream
	playlist   *Playlist
	downloader *Downloader
	reader     *Reader
	adCutter   *AdCutter
	streamer   *Streamer
	wg         *sync.WaitGroup
	stopFunc   func()
}

func (s *stream) Run(ctx context.Context) {
	ctx = log.With(ctx, "stream_id", s.stream.Id)
	ctx, s.stopFunc = context.WithCancel(ctx)
	s.playlist.Run(ctx, s.wg)
	s.downloader.Run(ctx, s.wg)
	s.reader.Run(ctx, s.wg)
	s.adCutter.Run(ctx, s.wg)
	s.streamer.Run(ctx, s.wg)
}

func (s *stream) Stop() {
	s.stopFunc()
	s.wg.Wait()
}
