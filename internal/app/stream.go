package app

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StreamType string

var (
	StreamTypePlaylist = "playlist"
)

type Stream struct {
	Id       int64
	Name     string
	IsActive bool
	Type     string
	Settings Settings
}

type Settings struct {
	Url          string `json:"url"`
	TgChannelId  int64  `json:"tg_channel_id"`
	PlaylistCode string `json:"playlist_code"`
}

func (y *Settings) Scan(v any) error {
	switch b := v.(type) {
	case []byte:
		return json.Unmarshal(b, y)
	default:
		return fmt.Errorf("unsupported type %T", v)
	}
}

func (y Settings) Value() (driver.Value, error) {
	b, err := json.Marshal(y)
	return string(b), err
}
