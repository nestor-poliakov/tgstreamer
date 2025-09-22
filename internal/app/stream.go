package app

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StreamType string

var (
	StreamTypePlaylist StreamType = "playlist"
	StreamTypeChannel  StreamType = "channel"
)

type Stream struct {
	Id       int64
	Name     string
	IsActive bool
	Type     StreamType
	Settings Settings
}

type Settings struct {
	Url         string `json:"url,omitempty"`
	TgChannelId int64  `json:"tg_channel_id,omitempty"`
	// youtube playlist id
	PlaylistCode string `json:"playlist_code,omitempty"`
	// youtube channel id
	ChannelCode string `json:"channel_code,omitempty"`
	//WxH
	Resolution   string `json:"resolution,omitempty"`
	AudioBitrate int    `json:"audio_bitrate,omitempty"`
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
