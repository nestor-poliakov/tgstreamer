package app

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
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
	PublishedAt int64
	Thumbnail   string
	Title       string
	Duration    time.Duration
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
	return json.Marshal(y)
}

type SponsorBlockInfo struct {
	Segments []Segment
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
	return json.Marshal(y)
}
