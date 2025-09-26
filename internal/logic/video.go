package logic

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/postgres"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
	"tgstreamer/lib/pg"
	"time"
)

type Video struct {
	youtube         *rpc.Youtube
	downloader      *rpc.YtDlpClient
	sb              *rpc.SponsorBlockClient
	videoStorage    postgres.Video
	toSetDownloaded chan infoUpdate
	wg              *sync.WaitGroup
	stopFunc        func()
}

func NewVideo(videoStorage postgres.Video, youtube *rpc.Youtube, downloader *rpc.YtDlpClient, sb *rpc.SponsorBlockClient) *Video {
	return &Video{
		youtube:         youtube,
		downloader:      downloader,
		sb:              sb,
		videoStorage:    videoStorage,
		toSetDownloaded: make(chan infoUpdate, 10),
		wg:              &sync.WaitGroup{},
		stopFunc:        func() {},
	}
}

func (v *Video) Run(ctx context.Context) {
	ctx, v.stopFunc = context.WithCancel(ctx)
	v.wg.Add(1)
	go v.settingDownloadedLoop(log.With(ctx, "worker", "video"))
	v.wg.Add(1)
	go v.gettingYtInfoLoop(log.With(ctx, "worker", "yt_info_downloader"))
	v.wg.Add(1)
	go v.gettingSbInfoLoop(log.With(ctx, "worker", "sb_info_downloader"))
	v.wg.Add(1)
	go v.deletingLoop(log.With(ctx, "worker", "file_deleter"))
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
			err := v.setDownloaded(ctx, downloaded)
			if err != nil {
				log.FromContext(ctx).Error("update video filename", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (v *Video) setDownloaded(ctx context.Context, downloaded infoUpdate) error {
	video, err := v.videoStorage.Get(ctx, downloaded.id)
	if err != nil {
		return fmt.Errorf("get video %d: %w", downloaded.id, err)
	}
	if video.FileInfo.Name == downloaded.info.Name && downloaded.info.Error == "" {
		return nil
	}
	err = v.videoStorage.AddFileInfo(ctx, video.Id, downloaded.info)
	if err != nil {
		return fmt.Errorf("update video file info: %w", err)
	}
	return nil
}

type infoUpdate struct {
	id   int64
	info app.FileInfo
}

func (v *Video) SetDownloaded(id int64, info app.FileInfo) {
	select {
	case v.toSetDownloaded <- infoUpdate{id: id, info: info}:
	default:
	}
}

func (v *Video) gettingYtInfoLoop(ctx context.Context) {
	defer v.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
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
}

func (v *Video) getYtInfo(ctx context.Context) error {
	videos, err := v.videoStorage.GetNoYtInfo(ctx)
	if err != nil {
		return fmt.Errorf("get video without youtube info: %w", err)
	}
	if len(videos) == 0 {
		return nil
	}
	codes := make([]string, len(videos))
	for i := range videos {
		codes[i] = videos[i].Code
	}
	infos, err := v.youtube.GetInfo(ctx, codes)
	if err != nil {
		return fmt.Errorf("get youtube info: %w", err)
	}
	for i := range videos {
		videos[i].YtInfo = infos[i]
	}
	err = v.videoStorage.AddYoutubeInfos(ctx, videos)
	if err != nil {
		return fmt.Errorf("add youtube info: %w", err)
	}
	log.FromContexts(ctx).Infof("youtube info for %d videos saved", len(videos))
	return nil
}

func (v *Video) gettingSbInfoLoop(ctx context.Context) {
	defer v.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := v.getSbInfo(ctx)
			if err != nil {
				log.FromContext(ctx).Error("get sb info", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (v *Video) getSbInfo(ctx context.Context) error {
	video, err := v.videoStorage.GetNoSbInfo(ctx)
	if errors.Is(err, pg.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get video without sponsor block info: %w", err)
	}
	video.SbInfo.Segments, err = v.sb.GetSegments(ctx, video.Code)
	if err != nil {
		return fmt.Errorf("get sponsor block info: %w", err)
	}
	video.SbInfo.LoadedAt = time.Now().Unix()
	err = v.videoStorage.AddSponsorBlockInfo(ctx, video.Id, video.SbInfo)
	if err != nil {
		return fmt.Errorf("add sponsor block info: %w", err)
	}
	log.FromContexts(ctx).Infof("sponsor block info for video %d saved", video.Id)
	return nil
}

func (v *Video) deletingLoop(ctx context.Context) {
	defer v.wg.Done()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := v.deleteFiles(ctx)
			if err != nil {
				log.FromContext(ctx).Error("delete files", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (v *Video) deleteFiles(ctx context.Context) error {
	files, err := v.downloader.GetFiles(ctx)
	if err != nil {
		return fmt.Errorf("get files: %w", err)
	}
	toDelete := make([]string, 0)
	for _, file := range files {
		if time.Since(file.ModAt) < time.Hour*24*7 {
			continue
		}
		toDelete = append(toDelete, file.Name)
	}
	log.FromContexts(ctx).Infof("found %d files to delete", len(toDelete))
	if len(toDelete) == 0 {
		return nil
	}
	err = v.downloader.DeleteFiles(ctx, toDelete)
	if err != nil {
		return fmt.Errorf("delete files: %w", err)
	}
	err = v.videoStorage.DeleteFileNames(ctx, toDelete)
	if err != nil {
		return fmt.Errorf("delete files names from storage: %w", err)
	}
	return nil
}
