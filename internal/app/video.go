package app

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Video struct {
	Id           int64
	Code         string
	FileName     string
	CreatedAt    int64
	DownloadedAt int64
	YtInfo       YoutubeInfo
	SbInfo       SponsorBlockInfo
}

type YoutubeInfo struct {
	LoadedAt    int64  `json:"loaded_at,omitempty"`
	PublishedAt int64  `json:"published_at,omitempty"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	Title       string `json:"title,omitempty"`
	// may be empty
	Duration int64 `json:"duration,omitempty"`
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

type PlaylistItem struct {
	Id       int64
	VideoId  int64
	StreamId int64
}
