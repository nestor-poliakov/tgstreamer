package rpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"tgstreamer/lib/log"
)

type YtDlpClient struct {
	filesDir string
}

func NewYtDlpClient(filesDir string) *YtDlpClient {
	y := &YtDlpClient{
		filesDir: filesDir,
	}
	return y
}

func (y *YtDlpClient) Download(ctx context.Context, videoUrl string) (string, error) {
	fileName := y.makeFileName(videoUrl)
	_, err := os.Stat(fileName)
	if err == nil {
		_ = exec.CommandContext(ctx, "touch", fileName).Run()
		log.FromContexts(ctx).Debugf("file %s already exist", fileName)
		return fileName, nil
	}
	cmd := exec.CommandContext(ctx, "yt-dlp", "-f", "mp4", "--geo-bypass", "-o", fileName, videoUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("run command %q: %w", cmd.String(), err)
	}
	return fileName, nil
}

func (y *YtDlpClient) makeFileName(videoUrl string) string {
	return path.Join(y.filesDir, strings.ReplaceAll(videoUrl, "/", "|")+".mp4")
}
