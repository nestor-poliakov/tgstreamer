package streamer

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
	"time"
)

type Manager struct {
	mu      sync.Mutex
	streams map[int64]*stream

	downloader    *rpc.YtDlpClient
	playlistLogic *logic.Playlist
	videoLogic    *logic.Video
	streamLogic   *logic.Stream
	playLogic     *logic.Play
	wg            *sync.WaitGroup
	stopFunc      func()
}

func NewManager(playlistLogic *logic.Playlist, videoLogic *logic.Video, streamLogic *logic.Stream, playLogic *logic.Play, downloader *rpc.YtDlpClient) *Manager {
	m := &Manager{
		streams:       make(map[int64]*stream),
		playlistLogic: playlistLogic,
		videoLogic:    videoLogic,
		streamLogic:   streamLogic,
		playLogic:     playLogic,
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
	m.wg.Add(1)
	go m.skipperLoop(ctx)
}

func (m *Manager) Stop() {
	m.stopFunc()
	m.wg.Wait()
}

func (m *Manager) skipperLoop(ctx context.Context) {
	ctx = log.With(ctx, "worker", "manager_skipper")
	defer log.FromContexts(ctx).Info("ended")
	defer m.wg.Done()
	ch := m.playLogic.GetToSkipChan()
	for {
		select {
		case <-ctx.Done():
			return
		case playToSkip := <-ch:
			err := m.skipPlay(ctx, playToSkip)
			if err != nil {
				log.FromContexts(ctx).With("error", err).Errorf("failed to skip play %v", playToSkip)
			}
		}
	}
}

func (m *Manager) skipPlay(ctx context.Context, play app.Play) error {
	log.FromContexts(ctx).Infof("skipping play %+v", play)
	playlistItem, err := m.playlistLogic.Get(ctx, play.PlaylistItemId)
	if err != nil {
		return fmt.Errorf("get playlist item: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	streamer, ok := m.streams[playlistItem.StreamId]
	if !ok {
		return fmt.Errorf("streamer %d not active", playlistItem.StreamId)
	}
	streamer.Skip(playlistItem.Id)
	return nil
}

func (m *Manager) processingLoop(ctx context.Context) {
	ctx = log.With(ctx, "worker", "manager_updater")
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
	log.FromContext(ctx).Info("start updating streams")
	streams, err := m.streamLogic.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get all streams: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	toStop := make([]int64, 0)
	toRun := make([]app.Stream, 0)
	streamsMap := map[int64]bool{}
	for _, stream := range streams {
		streamsMap[stream.Id] = true
	}
	for id := range m.streams {
		if !streamsMap[id] {
			toStop = append(toStop, id)
		}
	}
	for _, strm := range streams {
		runningStream, ok := m.streams[strm.Id]
		if !ok {
			toRun = append(toRun, strm)
			continue
		}
		if reflect.DeepEqual(runningStream.stream.Settings, strm.Settings) {
			continue
		}
		// settings changed, restart stream with new settings
		toStop = append(toStop, strm.Id)
		toRun = append(toRun, strm)
	}
	log.FromContexts(ctx).Infof("%d streams to stop; %d streams to start", len(toStop), len(toRun))
	for _, id := range toStop {
		m.stopStream(ctx, id)
	}
	for _, strm := range toRun {
		m.runStream(ctx, strm)
	}
	return nil
}

func (m *Manager) runStream(ctx context.Context, strm app.Stream) {
	s := newStream(strm, m.playlistLogic, m.videoLogic, m.playLogic, m.downloader)
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
