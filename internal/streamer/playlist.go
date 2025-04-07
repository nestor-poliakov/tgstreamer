package streamer

import (
	"context"
	"tgstreamer/internal/app"
)

func PlayVideos(ctx context.Context, streamingUrl string) {
	packetsCh := make(chan piece, 10)
	videos := make(chan app.Video, 0)
	NewReader(videos, packetsCh).Run(ctx)
	NewStreamer(streamingUrl, packetsCh).Run(ctx)
	videos <- app.Video{
		Id:       1,
		FileName: "/Users/nestor/go/src/tgstreamer/local/videos/https:||www.youtube.com|watch?v=qjxxYoL7nSU.mp4",
	}
	videos <- app.Video{
		Id:       2,
		FileName: "local/videos/video.flv",
	}
	videos <- app.Video{
		Id:       3,
		FileName: "local/videos/DeumyOzKqgI.flv",
	}
	videos <- app.Video{
		Id:       4,
		FileName: "local/videos/dQw4w9WgXcQ.flv",
	}
	<-ctx.Done()
	close(videos)
}
