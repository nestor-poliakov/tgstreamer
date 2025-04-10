package logic

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"tgstreamer/internal/postgres"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
	"tgstreamer/lib/pg"
	"time"
)

type Video struct {
	youtube         *rpc.Youtube
	videoStorage    postgres.Video
	toSetDownloaded chan downloaded
	wg              *sync.WaitGroup
	stopFunc        func()
}

func NewVideo(videoStorage postgres.Video, youtube *rpc.Youtube) *Video {
	return &Video{
		youtube:         youtube,
		videoStorage:    videoStorage,
		toSetDownloaded: make(chan downloaded, 10),
		wg:              &sync.WaitGroup{},
		stopFunc:        func() {},
	}
}

func (v *Video) Run(ctx context.Context) {
	ctx = log.With(ctx, "worker", "video")
	ctx, v.stopFunc = context.WithCancel(ctx)
	v.wg.Add(1)
	go v.settingDownloadedLoop(ctx)
	v.wg.Add(1)
	go v.gettingYtInfoLoop(ctx)
}

func (v *Video) Stop() {
	v.stopFunc()
	v.wg.Wait()
}

func (v *Video) settingDownloadedLoop(ctx context.Context) {
	defer v.wg.Done()
	for {
		select {
		case downloaded := <-v.toSetDownloaded:
			err := v.videoStorage.UpdateFileName(ctx, downloaded.id, downloaded.fileName)
			if err != nil {
				log.FromContext(ctx).Error("update video filename", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

type downloaded struct {
	id       int64
	fileName string
}

func (v *Video) SetDownloaded(id int64, fileName string) {
	select {
	case v.toSetDownloaded <- downloaded{id: id, fileName: fileName}:
	default:
	}
}

func (v *Video) gettingYtInfoLoop(ctx context.Context) {
	defer v.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		err := v.getYtInfo(ctx)
		if err != nil {
			log.FromContext(ctx).Error("get yt info", "error", err)
		}
	case <-ctx.Done():
		return
	}
}

func (v *Video) getYtInfo(ctx context.Context) error {
	video, err := v.videoStorage.GetNoYtInfo(ctx)
	if errors.Is(err, pg.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get video without youtube info: %w", err)
	}
	video.YtInfo, err = v.youtube.GetInfo(ctx, video.Code)
	if err != nil {
		return fmt.Errorf("get youtube info: %w", err)
	}
	err = v.videoStorage.AddYoutubeInfo(ctx, video.Id, video.YtInfo)
	if err != nil {
		return fmt.Errorf("add youtube info: %w", err)
	}
	log.FromContexts(ctx).Infof("youtube info for video %d saved", video.Id)
	return nil
}
