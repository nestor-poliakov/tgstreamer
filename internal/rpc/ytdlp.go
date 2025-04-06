package rpc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"tgstreamer/lib/log"
)

type YtDlpClient struct {
	filesDir string
	cmd      string
}

func NewYtDlpClient(filesDir string, cmd string) *YtDlpClient {
	y := &YtDlpClient{
		filesDir: filesDir,
		cmd:      cmd,
	}
	return y
}

func (y *YtDlpClient) Download(ctx context.Context, youtubeId string) (string, error) {
	fileName := path.Join(y.filesDir, youtubeId+".mp4")
	_, err := os.Stat(fileName)
	if err == nil {
		_ = exec.CommandContext(ctx, "touch", fileName).Run()
		log.FromContexts(ctx).Debugf("file %s already exist", fileName)
		return fileName, nil
	}
	cmd := exec.CommandContext(ctx, y.cmd, "-f", "mp4", "--geo-bypass", "-o", fileName, "https://www.youtube.com/watch?v="+youtubeId)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("run command %q: %w", cmd.String(), err)
	}
	return fileName, nil
}
