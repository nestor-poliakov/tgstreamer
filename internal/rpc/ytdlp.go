package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	stdErr := bytes.NewBuffer(nil)
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stdout, stdErr)
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("run command %q stderr: %s: %w", cmd.String(), stdErr.String(), err)
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

func (y *YtDlpClient) GetInfo(ctx context.Context, code string) {

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

type VideoInfo struct {
	Id                   string             `json:"id"`
	Title                string             `json:"title"`
	Formats              []Formats          `json:"formats"`
	Thumbnails           []Thumbnails       `json:"thumbnails"`
	Thumbnail            string             `json:"thumbnail"`
	Description          string             `json:"description"`
	ChannelId            string             `json:"channel_id"`
	ChannelUrl           string             `json:"channel_url"`
	Duration             int                `json:"duration"`
	ViewCount            int                `json:"view_count"`
	AverageRating        any                `json:"average_rating"`
	AgeLimit             int                `json:"age_limit"`
	WebpageUrl           string             `json:"webpage_url"`
	Categories           []string           `json:"categories"`
	Tags                 []string           `json:"tags"`
	PlayableInEmbed      bool               `json:"playable_in_embed"`
	LiveStatus           string             `json:"live_status"`
	MediaType            any                `json:"media_type"`
	ReleaseTimestamp     any                `json:"release_timestamp"`
	FormatSortFields     []string           `json:"_format_sort_fields"`
	AutomaticCaptions    AutomaticCaptions  `json:"automatic_captions"`
	Subtitles            Subtitles          `json:"subtitles"`
	CommentCount         int                `json:"comment_count"`
	Chapters             any                `json:"chapters"`
	Heatmap              any                `json:"heatmap"`
	LikeCount            int                `json:"like_count"`
	Channel              string             `json:"channel"`
	ChannelFollowerCount int                `json:"channel_follower_count"`
	Uploader             string             `json:"uploader"`
	UploaderId           string             `json:"uploader_id"`
	UploaderURL          string             `json:"uploader_url"`
	UploadDate           string             `json:"upload_date"`
	Timestamp            int                `json:"timestamp"`
	Availability         string             `json:"availability"`
	OriginalURL          string             `json:"original_url"`
	WebpageUrlBasename   string             `json:"webpage_url_basename"`
	WebpageUrlDomain     string             `json:"webpage_url_domain"`
	Extractor            string             `json:"extractor"`
	ExtractorKey         string             `json:"extractor_key"`
	Playlist             any                `json:"playlist"`
	PlaylistIndex        any                `json:"playlist_index"`
	DisplayId            string             `json:"display_id"`
	Fulltitle            string             `json:"fulltitle"`
	DurationString       string             `json:"duration_string"`
	ReleaseYear          any                `json:"release_year"`
	IsLive               bool               `json:"is_live"`
	WasLive              bool               `json:"was_live"`
	RequestedSubtitles   any                `json:"requested_subtitles"`
	HasDrm               any                `json:"_has_drm"`
	Epoch                int                `json:"epoch"`
	RequestedFormats     []RequestedFormats `json:"requested_formats"`
	Format               string             `json:"format"`
	FormatId             string             `json:"format_id"`
	Ext                  string             `json:"ext"`
	Protocol             string             `json:"protocol"`
	Language             any                `json:"language"`
	FormatNote           string             `json:"format_note"`
	FilesizeApprox       int                `json:"filesize_approx"`
	Tbr                  float64            `json:"tbr"`
	Width                int                `json:"width"`
	Height               int                `json:"height"`
	Resolution           string             `json:"resolution"`
	Fps                  int                `json:"fps"`
	DynamicRange         string             `json:"dynamic_range"`
	Vcodec               string             `json:"vcodec"`
	Vbr                  float64            `json:"vbr"`
	StretchedRatio       any                `json:"stretched_ratio"`
	AspectRatio          float64            `json:"aspect_ratio"`
	Acodec               string             `json:"acodec"`
	Abr                  float64            `json:"abr"`
	Asr                  int                `json:"asr"`
	AudioChannels        int                `json:"audio_channels"`
	Filename             string             `json:"_filename"`
	Filename0            string             `json:"filename"`
	Type                 string             `json:"_type"`
	Version              Version            `json:"_version"`
}
type Fragments struct {
	Url      string  `json:"url"`
	Duration float64 `json:"duration"`
}
type HTTPHeaders struct {
	UserAgent      string `json:"User-Agent"`
	Accept         string `json:"Accept"`
	AcceptLanguage string `json:"Accept-Language"`
	SecFetchMode   string `json:"Sec-Fetch-Mode"`
}
type DownloaderOptions struct {
	HttpChunkSize int `json:"http_chunk_size"`
}
type Formats struct {
	FormatId           string            `json:"format_id"`
	FormatNote         string            `json:"format_note,omitempty"`
	Ext                string            `json:"ext"`
	Protocol           string            `json:"protocol"`
	Acodec             string            `json:"acodec,omitempty"`
	Vcodec             string            `json:"vcodec"`
	Url                string            `json:"url"`
	Width              int               `json:"width,omitempty"`
	Height             int               `json:"height,omitempty"`
	Fps                float64           `json:"fps,omitempty"`
	Rows               int               `json:"rows,omitempty"`
	Columns            int               `json:"columns,omitempty"`
	Fragments          []Fragments       `json:"fragments,omitempty"`
	AudioExt           string            `json:"audio_ext"`
	VideoExt           string            `json:"video_ext"`
	Vbr                int               `json:"vbr"`
	Abr                int               `json:"abr"`
	Tbr                any               `json:"tbr"`
	Resolution         string            `json:"resolution"`
	AspectRatio        float64           `json:"aspect_ratio"`
	FilesizeApprox     any               `json:"filesize_approx,omitempty"`
	HttpHeaders        HTTPHeaders       `json:"http_headers"`
	Format             string            `json:"format"`
	FormatIndex        any               `json:"format_index,omitempty"`
	ManifestUrl        string            `json:"manifest_url,omitempty"`
	Language           any               `json:"language,omitempty"`
	Preference         any               `json:"preference,omitempty"`
	Quality            int               `json:"quality,omitempty"`
	HasDrm             bool              `json:"has_drm,omitempty"`
	SourcePreference   int               `json:"source_preference,omitempty"`
	Asr                int               `json:"asr,omitempty"`
	Filesize           int               `json:"filesize,omitempty"`
	AudioChannels      int               `json:"audio_channels,omitempty"`
	LanguagePreference int               `json:"language_preference,omitempty"`
	DynamicRange       any               `json:"dynamic_range,omitempty"`
	Container          string            `json:"container,omitempty"`
	DownloaderOptions  DownloaderOptions `json:"downloader_options,omitempty"`
}
type Thumbnails struct {
	Url        string `json:"url"`
	Preference int    `json:"preference"`
	Id         string `json:"id"`
	Height     int    `json:"height,omitempty"`
	Width      int    `json:"width,omitempty"`
	Resolution string `json:"resolution,omitempty"`
}
type AutomaticCaptions struct{}

type Subtitles struct{}

type RequestedFormats struct {
	Asr                any               `json:"asr"`
	Filesize           int               `json:"filesize"`
	FormatId           string            `json:"format_id"`
	FormatNote         string            `json:"format_note"`
	SourcePreference   int               `json:"source_preference"`
	Fps                int               `json:"fps"`
	AudioChannels      any               `json:"audio_channels"`
	Height             int               `json:"height"`
	Quality            float64           `json:"quality"`
	HasDrm             bool              `json:"has_drm"`
	Tbr                float64           `json:"tbr"`
	FilesizeApprox     int               `json:"filesize_approx"`
	Url                string            `json:"url"`
	Width              int               `json:"width"`
	Language           any               `json:"language"`
	LanguagePreference int               `json:"language_preference"`
	Preference         any               `json:"preference"`
	Ext                string            `json:"ext"`
	Vcodec             string            `json:"vcodec"`
	Acodec             string            `json:"acodec"`
	DynamicRange       string            `json:"dynamic_range"`
	Container          string            `json:"container"`
	DownloaderOptions  DownloaderOptions `json:"downloader_options"`
	Protocol           string            `json:"protocol"`
	VideoExt           string            `json:"video_ext"`
	AudioExt           string            `json:"audio_ext"`
	Abr                int               `json:"abr"`
	Vbr                float64           `json:"vbr"`
	Resolution         string            `json:"resolution"`
	AspectRatio        float64           `json:"aspect_ratio"`
	HttpHeaders        HTTPHeaders       `json:"http_headers"`
	Format             string            `json:"format"`
}
type Version struct {
	Version        string `json:"version"`
	CurrentGitHead any    `json:"current_git_head"`
	ReleaseGitHead string `json:"release_git_head"`
	Repository     string `json:"repository"`
}
