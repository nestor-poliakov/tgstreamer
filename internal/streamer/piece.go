package streamer

import (
	"tgstreamer/internal/app"

	"github.com/nestor-poliakov/joy5/av"
)

// group of packets started with key frame
type piece struct {
	playlistItemId int64
	packets        []av.Packet
	video          app.Video
}

type video struct {
	playlistItemId int64
	video          app.Video
}
