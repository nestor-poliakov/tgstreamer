package rpc

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"tgstreamer/internal/app"
	"tgstreamer/lib/log"
)

type DownloadedSetter interface {
	SetDownloaded(id int64, fileName string)
}

type YtDlpClient struct {
	setter   DownloadedSetter
	filesDir string
}

func NewYtDlpClient(setter DownloadedSetter, filesDir string) *YtDlpClient {
	y := &YtDlpClient{
		setter:   setter,
		filesDir: filesDir,
	}
	return y
}

func (y *YtDlpClient) DownloadYt(ctx context.Context, video app.Video) (string, error) {
	return y.Download(ctx, video.Id, "https://www.youtube.com/watch?"+url.Values{"v": {video.Code}}.Encode())
}

func (y *YtDlpClient) Download(ctx context.Context, videoId int64, videoUrl string) (string, error) {
	log.FromContexts(ctx).Info("start downloading video " + videoUrl)
	fileName := y.makeFileName(videoUrl)
	_, err := os.Stat(fileName)
	if err == nil {
		_ = exec.CommandContext(ctx, "touch", fileName).Run()
		log.FromContexts(ctx).Debugf("file %s already exist", fileName)
		return fileName, nil
	}
	t := time.Now()
	cmd := exec.CommandContext(ctx, "yt-dlp", "-f", "mp4", "--geo-bypass", "-o", fileName, videoUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("run command %q: %w", cmd.String(), err)
	}
	info, err := os.Stat(fileName)
	if err != nil {
		return "", fmt.Errorf("stat file %q: %w", fileName, err)
	}
	y.setter.SetDownloaded(videoId, fileName)
	log.FromContexts(ctx).Infof("%dMB downloaded for %s", info.Size()/1024/1024, time.Since(t))
	return fileName, nil
}

func (y *YtDlpClient) makeFileName(videoUrl string) string {
	return path.Join(y.filesDir, strings.ReplaceAll(videoUrl, "/", "|")+".mp4")
}
