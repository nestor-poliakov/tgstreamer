package app

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Video struct {
	Id int64 `db:"id"`
	// youtube video id
	Code      string           `db:"code"`
	CreatedAt int64            `db:"created_at"`
	FileInfo  FileInfo         `db:"file_info"`
	YtInfo    YoutubeInfo      `db:"yt_info"`
	SbInfo    SponsorBlockInfo `db:"sb_info"`
}

type FileInfo struct {
	Name          string `json:"name,omitempty"`
	Error         string `json:"error,omitempty"`
	DownloadedAt  int64  `json:"downloaded_at,omitempty"`
	AudioChannels int    `json:"audio_channels,omitempty"`
	Size          int    `json:"size,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	DurationA     int    `json:"duration_a,omitempty"`
	DurationV     int    `json:"duration_v,omitempty"`
	AudioBitrate  int    `json:"audio_bitrate,omitempty"`
	StartAudio    int    `json:"start_audio,omitempty"`
	StartVideo    int    `json:"start_video,omitempty"`
	EndAudio      int    `json:"end_audio,omitempty"`
	EndVideo      int    `json:"end_video,omitempty"`
	Fps           int    `json:"fps,omitempty"`
}

type YoutubeInfo struct {
	LoadedAt    int64  `json:"loaded_at,omitempty"`
	PublishedAt int64  `json:"published_at,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	Title       string `json:"title,omitempty"`
}

type SponsorBlockInfo struct {
	LoadedAt int64     `json:"loadedAt"`
	Segments []Segment `json:"segments"`
}

type Segment struct {
	Id            string     `json:"UUID"`
	Segment       [2]float64 `json:"segment"`
	Category      string     `json:"category"`
	ActionType    string     `json:"actionType"`
	Locked        int        `json:"locked"`
	Votes         int        `json:"votes"`
	VideoDuration float64    `json:"videoDuration"`
	Description   string     `json:"description"`
}

type PlaylistItem struct {
	Id       int64 `db:"id"`
	VideoId  int64 `db:"video_id"`
	StreamId int64 `db:"stream_id"`
}

func (y *YoutubeInfo) Scan(v any) error {
	switch b := v.(type) {
	case []byte:
		return json.Unmarshal(b, y)
	default:
		return fmt.Errorf("unsupported type %T", v)
	}
}

func (y YoutubeInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(y)
	return string(b), err
}

func (y *SponsorBlockInfo) Scan(v any) error {
	switch b := v.(type) {
	case []byte:
		return json.Unmarshal(b, y)
	default:
		return fmt.Errorf("unsupported type %T", v)
	}
}

func (y SponsorBlockInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(y)
	return string(b), err
}

func (y *FileInfo) Scan(v any) error {
	switch b := v.(type) {
	case []byte:
		return json.Unmarshal(b, y)
	default:
		return fmt.Errorf("unsupported type %T", v)
	}
}

func (y FileInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(y)
	return string(b), err
}
