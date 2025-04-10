package logic

import (
	"context"
	"sync"
	"tgstreamer/internal/postgres"
	"tgstreamer/lib/log"
)

type Video struct {
	videoStorage    postgres.Video
	toSetDownloaded chan downloaded
	wg              *sync.WaitGroup
	stopFunc        func()
}

func NewVideo(videoStorage postgres.Video) *Video {
	return &Video{
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
	go v.processingLoop(ctx)
}

func (v *Video) Stop() {
	v.stopFunc()
	v.wg.Wait()
}

func (v *Video) processingLoop(ctx context.Context) {
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
