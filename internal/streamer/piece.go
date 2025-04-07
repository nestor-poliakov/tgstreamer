package streamer

import "github.com/nestor-poliakov/joy5/av"

type piece struct {
	videoId int64
	packets []av.Packet
}
