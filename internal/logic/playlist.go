package logic

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/postgres"
	"tgstreamer/lib/log"
	"tgstreamer/lib/pg"
)

type Playlist struct {
	playlistStorage postgres.Playlist
	videoStorage    postgres.Video
	toSetCurrent    chan int64
	wg              *sync.WaitGroup
	stopFunc        func()
}

func NewPlaylist(playlist postgres.Playlist, video postgres.Video) *Playlist {
	p := &Playlist{
		playlistStorage: playlist,
		videoStorage:    video,
		toSetCurrent:    make(chan int64, 10),
		wg:              &sync.WaitGroup{},
		stopFunc:        func() {},
	}
	return p
}

func (p *Playlist) Run(ctx context.Context) {
	ctx = log.With(ctx, "worker", "playlist")
	ctx, p.stopFunc = context.WithCancel(ctx)
	p.wg.Add(1)
	go p.processingLoop(ctx)
}

func (p *Playlist) Stop() {
	p.stopFunc()
	p.wg.Wait()
}

func (p *Playlist) processingLoop(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case id := <-p.toSetCurrent:
			err := p.setCurrent(ctx, id)
			if err != nil {
				log.FromContext(ctx).Error("set current playlist item", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *Playlist) setCurrent(ctx context.Context, id int64) error {
	return p.playlistStorage.SetCurrent(ctx, id)
}

func (p *Playlist) GetCurrent(ctx context.Context, streamId int64) (int64, app.Video, error) {
	pliId, videoId, err := p.playlistStorage.GetCurrent(ctx, streamId)
	if errors.Is(err, pg.ErrNoRows) {
		return p.getFirst(ctx, streamId)
	}
	if err != nil {
		return 0, app.Video{}, fmt.Errorf("get current playlist item: %w", err)
	}
	video, err := p.videoStorage.Get(ctx, videoId)
	if err != nil {
		return 0, app.Video{}, fmt.Errorf("get video: %w", err)
	}
	return pliId, video, nil
}

func (p *Playlist) GetNext(ctx context.Context, pliId int64) (int64, app.Video, error) {
	newPliId, videoId, err := p.playlistStorage.GetNext(ctx, pliId)
	if errors.Is(err, pg.ErrNoRows) {
		streamId, err := p.playlistStorage.GetStreamId(ctx, pliId)
		if err != nil {
			return 0, app.Video{}, fmt.Errorf("get stream id for playlist item: %w", err)
		}
		return p.getFirst(ctx, streamId)
	}
	if err != nil {
		return 0, app.Video{}, fmt.Errorf("get next playlist item: %w", err)
	}
	video, err := p.videoStorage.Get(ctx, videoId)
	if err != nil {
		return 0, app.Video{}, fmt.Errorf("get video: %w", err)
	}
	return newPliId, video, nil
}

func (p *Playlist) getFirst(ctx context.Context, streamId int64) (int64, app.Video, error) {
	pliId, videoId, err := p.playlistStorage.GetFirst(ctx, streamId)
	if err != nil {
		return 0, app.Video{}, fmt.Errorf("get first playlist item: %w", err)
	}
	video, err := p.videoStorage.Get(ctx, videoId)
	if err != nil {
		return 0, app.Video{}, fmt.Errorf("get video: %w", err)
	}
	log.FromContexts(ctx).Debugf("first playlist item: %d; video: %v", pliId, video)
	return pliId, video, nil
}

func (p *Playlist) SetCurrent(id int64) {
	select {
	case p.toSetCurrent <- id:
	default:
	}
}
