package logic

import (
	"context"
	"fmt"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/postgres"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
)

type Play struct {
	tg       *rpc.Telegram
	playlist postgres.Playlist
	play     postgres.Play
	video    postgres.Video
	stream   postgres.Stream

	toSkip     chan app.Play
	toAnnounce chan int64
	stopFunc   func()
	wg         sync.WaitGroup
}

func NewPlay(play postgres.Play, stream postgres.Stream, video postgres.Video, playlist postgres.Playlist, tg *rpc.Telegram) *Play {
	return &Play{
		tg:         tg,
		playlist:   playlist,
		play:       play,
		video:      video,
		stream:     stream,
		toSkip:     make(chan app.Play, 10),
		toAnnounce: make(chan int64, 10),
		stopFunc:   func() {},
	}
}

func (p *Play) Run(ctx context.Context) {
	ctx, p.stopFunc = context.WithCancel(ctx)
	p.wg.Add(1)
	go p.processingLoop(ctx)
}

func (p *Play) Stop() {
	p.stopFunc()
	p.wg.Wait()
}

func (p *Play) processingLoop(ctx context.Context) {
	ctx = log.With(ctx, "worker", "announcer")
	defer p.wg.Done()
	for {
		select {
		case playlistItemId := <-p.toAnnounce:
			err := p.announce(ctx, playlistItemId)
			if err != nil {
				log.Defaults().Errorf("failed to announce playlist item %d: %v", playlistItemId, err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *Play) Like(ctx context.Context, playId int64) error {
	_, err := p.play.IncLike(ctx, playId)
	if err != nil {
		return fmt.Errorf("add like: %w", err)
	}
	return nil
}

func (p *Play) Skip(ctx context.Context, playId int64) error {
	play, err := p.play.IncSkip(ctx, playId)
	if err != nil {
		return fmt.Errorf("add skip: %w", err)
	}
	if play.Skips > 10 {
		select {
		case p.toSkip <- play:
		default:
		}

	}
	return nil
}

func (p *Play) Announce(playlistItemId int64) {
	select {
	case p.toAnnounce <- playlistItemId:
	default:
		log.Defaults().Errorf("failed to announce playlist item %d", playlistItemId)
	}
}

func (p *Play) announce(ctx context.Context, playlistItemId int64) error {
	playlistItem, err := p.playlist.Get(ctx, playlistItemId)
	if err != nil {
		return fmt.Errorf("get playlist item: %w", err)
	}
	video, err := p.video.Get(ctx, playlistItem.Id)
	if err != nil {
		return fmt.Errorf("get video: %w", err)
	}
	stream, err := p.stream.Get(ctx, playlistItem.StreamId)
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}
	play, err := p.play.Create(ctx, app.Play{
		PlaylistItemId: playlistItemId,
	})
	if err != nil {
		return fmt.Errorf("create play: %w", err)
	}
	msgId, err := p.tg.Announce(ctx, stream.Settings, video, play.Id)
	if err != nil {
		return fmt.Errorf("announce video: %w", err)
	}
	err = p.play.AddMsgId(ctx, play.Id, msgId)
	if err != nil {
		return fmt.Errorf("add message id to play: %w", err)
	}
	return nil
}

func (p *Play) GetToSkipChan() <-chan app.Play {
	return p.toSkip
}
