package streamer

import (
	"context"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/lib/log"
	"time"
)

type Playlist struct {
	playlistLogic *logic.Playlist
	stream        app.Stream
	toReader      chan<- video
}

func NewPlaylist(toReader chan<- video, s app.Stream, playlistLogic *logic.Playlist) *Playlist {
	pl := &Playlist{
		playlistLogic: playlistLogic,
		stream:        s,
		toReader:      toReader,
	}
	return pl
}

func (m *Playlist) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx = log.With(ctx, "worker", "playlist")
	wg.Add(1)
	go m.processingLoop(ctx, wg)
}

func (m *Playlist) processingLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(m.toReader)
	plii, video := m.getCurrent(ctx)
	log.FromContexts(ctx).Debugf("current playlist item: %d", plii)
	for {
		err := m.processVideo(ctx, video, plii)
		if err != nil {
			return
		}
		plii, video = m.getNext(ctx, plii)
		log.FromContexts(ctx).Debugf("next playlist item: %d", plii)
	}
}

func (m *Playlist) processVideo(ctx context.Context, v app.Video, plii int64) error {
	if v.FileInfo.Error != "" {
		log.FromContexts(ctx).With("error", v.FileInfo.Error, "video_id", v.Id).Info("not processing video with error")
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.toReader <- video{
		playlistItemId: plii,
		video:          v,
	}:
		return nil
	}
}

func (m *Playlist) getCurrent(ctx context.Context) (int64, app.Video) {
	for {
		id, video, err := m.playlistLogic.GetCurrent(ctx, m.stream.Id)
		if err != nil {
			log.FromContext(ctx).Error("get current video", "error", err)
			select {
			case <-ctx.Done():
				return 0, app.Video{}
			case <-time.After(time.Minute):
				continue
			}
		}
		return id, video
	}
}

func (m *Playlist) getNext(ctx context.Context, id int64) (int64, app.Video) {
	for {
		nextId, video, err := m.playlistLogic.GetNext(ctx, id)
		if err != nil {
			log.FromContext(ctx).Error("get next video", "error", err)
			select {
			case <-ctx.Done():
				return 0, app.Video{}
			case <-time.After(time.Minute):
				continue
			}
		}
		return nextId, video
	}
}
