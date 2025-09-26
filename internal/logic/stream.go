package logic

import (
	"context"
	"fmt"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/postgres"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
	"time"
)

type Stream struct {
	streamStorage   postgres.Stream
	videoStorage    postgres.Video
	playlistStorage postgres.Playlist
	youtube         *rpc.Youtube

	wg       *sync.WaitGroup
	stopFunc context.CancelFunc
}

func NewStream(streamStorage postgres.Stream, videoStorage postgres.Video, playlistStorage postgres.Playlist, youtube *rpc.Youtube) *Stream {
	return &Stream{
		streamStorage:   streamStorage,
		videoStorage:    videoStorage,
		playlistStorage: playlistStorage,
		youtube:         youtube,
		wg:              &sync.WaitGroup{},
		stopFunc:        func() {},
	}
}

func (s *Stream) Run(ctx context.Context) {
	ctx, s.stopFunc = context.WithCancel(ctx)
	s.wg.Add(1)
	go s.playlistUpdateLoop(log.With(ctx, "worker", "playlist_updater"))
}

func (s *Stream) Stop() {
	s.stopFunc()
	s.wg.Wait()
}

func (s *Stream) playlistUpdateLoop(ctx context.Context) {
	defer s.wg.Done()
	t := time.NewTicker(time.Hour * 10)
	defer t.Stop()
	err := s.updatePlaylistsInfo(ctx)
	if err != nil {
		log.FromContext(ctx).Error("update playlist info", "error", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := s.updatePlaylistsInfo(ctx)
			if err != nil {
				log.FromContext(ctx).Error("update playlist info", "error", err)
			}
		}
	}
}

func (s *Stream) updatePlaylistsInfo(ctx context.Context) error {
	streams, err := s.streamStorage.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get active streams: %w", err)
	}
	log.FromContexts(ctx).Infof("%d streams to update", len(streams))
	for _, stream := range streams {
		ctx := log.With(ctx, "stream_id", stream.Id)
		err := s.updatePlaylistInfo(ctx, stream)
		if err != nil {
			return fmt.Errorf("update playlist info: %w", err)
		}
		time.Sleep(time.Second)
	}
	return nil
}

func (s *Stream) updatePlaylistInfo(ctx context.Context, stream app.Stream) (err error) {
	switch stream.Type {
	case app.StreamTypePlaylist:
		return s.updatePlaylistForPlaylist(ctx, stream)
	case app.StreamTypeChannel:
		return s.updatePlaylistForChannel(ctx, stream)
	default:
		return fmt.Errorf("unknown stream type %q", stream.Type)
	}
}

func (s *Stream) updatePlaylistForChannel(ctx context.Context, stream app.Stream) (err error) {
	if stream.Settings.ChannelCode == "" {
		return fmt.Errorf("channel code is empty")
	}
	videos, err := s.youtube.GetChannelVideos(ctx, stream.Settings.ChannelCode)
	if err != nil {
		return fmt.Errorf("get channel items from youtube: %w", err)
	}
	log.FromContexts(ctx).Infof("%d videos found in youtube channel", len(videos))
	videos, err = s.videoStorage.CreateList(ctx, videos)
	if err != nil {
		return fmt.Errorf("create videos: %w", err)
	}
	curPlaylist, err := s.playlistStorage.GetForStream(ctx, stream.Id)
	if err != nil {
		return fmt.Errorf("get playlist for stream: %w", err)
	}
	itemsToCreate := s.makeItemsNotInPlaylist(curPlaylist, videos)
	log.FromContexts(ctx).Infof("%d items to add to playlist", len(itemsToCreate))
	err = s.playlistStorage.CreateList(ctx, itemsToCreate, stream.Id)
	if err != nil {
		return fmt.Errorf("add videos to playlist: %w", err)
	}
	return nil
}

func (s *Stream) updatePlaylistForPlaylist(ctx context.Context, stream app.Stream) (err error) {
	if stream.Settings.PlaylistCode == "" {
		return fmt.Errorf("playlist code is empty")
	}
	videos, err := s.youtube.GetPlaylist(ctx, stream.Settings.PlaylistCode)
	if err != nil {
		return fmt.Errorf("get playlist items from youtube: %w", err)
	}
	log.FromContexts(ctx).Infof("%d videos found in youtube playlist", len(videos))
	videos, err = s.videoStorage.CreateList(ctx, videos)
	if err != nil {
		return fmt.Errorf("create videos: %w", err)
	}
	curPlaylist, err := s.playlistStorage.GetForStream(ctx, stream.Id)
	if err != nil {
		return fmt.Errorf("get playlist for stream: %w", err)
	}
	itemsToCreate := s.makeItemsNotInPlaylist(curPlaylist, videos)
	log.FromContexts(ctx).Infof("%d items to add to playlist", len(itemsToCreate))
	err = s.playlistStorage.CreateList(ctx, itemsToCreate, stream.Id)
	if err != nil {
		return fmt.Errorf("add videos to playlist: %w", err)
	}
	return nil
}

func (s *Stream) makeItemsNotInPlaylist(playlist []app.PlaylistItem, videos []app.Video) (res []int64) {
	playlistMap := make(map[int64]bool, len(playlist))
	for _, item := range playlist {
		playlistMap[item.VideoId] = true
	}

	for _, video := range videos {
		_, ok := playlistMap[video.Id]
		if !ok {
			res = append(res, video.Id)
		}
	}
	return res
}

func (s *Stream) GetActive(ctx context.Context) ([]app.Stream, error) {
	return s.streamStorage.GetActive(ctx)
}
