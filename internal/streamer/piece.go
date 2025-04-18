package streamer

import (
	"tgstreamer/internal/app"

	"github.com/nestor-poliakov/joy5/av"
)

type piece struct {
	Video   *app.Video
	videoId int64
	packets []av.Packet
}
