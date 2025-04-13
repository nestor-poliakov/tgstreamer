package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"time"

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

func (y *YtDlpClient) DownloadYt(ctx context.Context, videoCode string) (string, error) {
	return y.download(ctx, "https://www.youtube.com/watch?"+url.Values{"v": {videoCode}}.Encode(), y.makeFileName(videoCode))
}

func (y *YtDlpClient) Touch(ctx context.Context, videoCode string) {
	fileName := y.makeFileName(videoCode)
	y.touch(ctx, fileName)
}

func (y *YtDlpClient) download(ctx context.Context, videoUrl string, fileName string) (string, error) {
	_, err := os.Stat(fileName)
	if err == nil {
		log.FromContexts(ctx).Infof("file %q already exist, touching", fileName)
		y.touch(ctx, fileName)
		return fileName, nil
	}
	log.FromContexts(ctx).Info("start downloading video " + videoUrl)
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
	log.FromContexts(ctx).Infof("%dMB downloaded for %s", info.Size()/1024/1024, time.Since(t))
	y.touch(ctx, fileName)
	return fileName, nil
}

func (y *YtDlpClient) makeFileName(videoCode string) string {
	return path.Join(y.filesDir, videoCode+".mp4")
}

func (y YtDlpClient) touch(ctx context.Context, fileName string) {
	err := os.Chtimes(fileName, time.Now(), time.Now())
	if err != nil {
		log.FromContexts(ctx).Errorf("failed to touch file %q: %v", fileName, err)
	}
}

func (y *YtDlpClient) GetFiles(ctx context.Context) ([]File, error) {
	files, err := os.ReadDir(y.filesDir)
	if err != nil {
		return nil, fmt.Errorf("read %s dir: %w", y.filesDir, err)
	}
	res := make([]File, 0, len(files))
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			return nil, fmt.Errorf("stat file %q: %w", f.Name(), err)
		}
		if info.IsDir() {
			continue
		}

		res = append(res, File{
			Name:  path.Join(y.filesDir, f.Name()),
			ModAt: info.ModTime(),
		})
	}
	return res, nil
}

type File struct {
	Name  string
	ModAt time.Time
}

func (y *YtDlpClient) DeleteFiles(ctx context.Context, fileNames []string) error {
	var errs []error
	for _, fileName := range fileNames {
		err := os.Remove(fileName)
		if err != nil {
			errs = append(errs, fmt.Errorf("remove file %q: %w", fileName, err))
		}
	}
	return errors.Join(errs...)
}
