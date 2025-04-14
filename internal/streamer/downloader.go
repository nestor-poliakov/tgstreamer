package streamer

import (
	"context"
	"fmt"
	"os"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"

	"github.com/yapingcat/gomedia/go-mp4"
)

type Downloader struct {
	resolution string
	downloader *rpc.YtDlpClient
	videoLogic *logic.Video
	from       <-chan app.Video
	to         chan<- app.Video
}

func NewDownloader(from <-chan app.Video, to chan<- app.Video, downloader *rpc.YtDlpClient, videoLogic *logic.Video, resolution string) *Downloader {
	return &Downloader{
		from:       from,
		to:         to,
		downloader: downloader,
		videoLogic: videoLogic,
		resolution: resolution,
	}
}

func (d *Downloader) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	ctx = log.With(ctx, "worker", "downloader")
	go d.downloadingLoop(ctx, wg)
}

func (d *Downloader) downloadingLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(d.to)
	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-d.from:
			if !ok {
				return
			}
			vctx := log.With(ctx, "video", v.Id)
			v, err := d.download(vctx, v)
			if err != nil {
				log.FromContext(vctx).Error("download video", "error", err)
				continue
			}
			select {
			case d.to <- v:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (d *Downloader) download(ctx context.Context, video app.Video) (app.Video, error) {
	fileName, err := d.downloader.DownloadYt(ctx, video.Code)
	if err != nil {
		return video, fmt.Errorf("download video: %w", err)
	}
	video.FileInfo, err = d.getInfo(ctx, fileName)
	if err != nil {
		return video, fmt.Errorf("get info: %w", err)
	}
	d.videoLogic.SetDownloaded(video.Id, video.FileInfo)
	if d.resolution != "" && d.resolution != fmt.Sprintf("%dx%d", video.FileInfo.Width, video.FileInfo.Height) {
		return video, fmt.Errorf("resolution not allowed")
	}
	return video, nil
}

func (d *Downloader) getInfo(ctx context.Context, fileName string) (info app.FileInfo, err error) {
	info.Name = fileName
	f, err := os.Open(fileName)
	if err != nil {
		return info, fmt.Errorf("open file: %w", err)
	}
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return info, fmt.Errorf("stat file: %w", err)
	}
	info.Size = int(fileInfo.Size())
	defer f.Close()
	tracks, err := mp4.CreateMp4Demuxer(f).ReadHead()
	if err != nil {
		return info, fmt.Errorf("read head: %w", err)
	}
	for _, track := range tracks {
		switch track.Cid {
		case mp4.MP4_CODEC_AAC:
			info.AudioChannels = int(track.ChannelCount)
		case mp4.MP4_CODEC_H264:
			info.Width = int(track.Width)
			info.Height = int(track.Height)
			if track.Duration > 0 && track.Timescale > 0 {
				durationSec := float64(track.Duration) / float64(track.Timescale)
				info.Fps = int(float64(track.SampleCount) / durationSec)
				info.Duration = int(durationSec)
			}
		}
	}
	return info, nil
}
