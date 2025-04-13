package streamer

import (
	"context"
	"fmt"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
)

type Downloader struct {
	downloader *rpc.YtDlpClient
	videoLogic *logic.Video
	from       <-chan app.Video
	to         chan<- app.Video
}

func NewDownloader(from <-chan app.Video, to chan<- app.Video, downloader *rpc.YtDlpClient, videoLogic *logic.Video) *Downloader {
	return &Downloader{
		from:       from,
		to:         to,
		downloader: downloader,
		videoLogic: videoLogic,
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
				log.FromContext(vctx).Error("msg string")
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
	var err error
	video.FileName, err = d.downloader.DownloadYt(ctx, video.Code)
	if err != nil {
		return video, fmt.Errorf("download video: %w", err)
	}
	d.videoLogic.SetDownloaded(video.Id, video.FileName)
	return video, nil
}
