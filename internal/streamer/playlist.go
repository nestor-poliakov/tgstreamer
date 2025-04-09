package streamer

import (
	"context"
	"tgstreamer/internal/app"
	"tgstreamer/internal/rpc"
)

func PlayVideos(ctx context.Context, config app.Config) {
	packetsCh := make(chan piece, 10)
	videos := make(chan app.Video, 10)
	ytDlpClient := rpc.NewYtDlpClient(config.VideosDir)
	NewReader(videos, packetsCh, ytDlpClient).Run(ctx)
	NewStreamer(config.StreamingUrl, packetsCh).Run(ctx)
	videos <- app.Video{
		Id:  101,
		Url: "https://www.youtube.com/watch?v=4evV8Fr5A8U",
	}
	videos <- app.Video{
		Id:  1,
		Url: "https://www.youtube.com/watch?v=8OkpRK2_gVs",
	}

	videos <- app.Video{
		Id:  2,
		Url: "https://www.youtube.com/watch?v=jIfogFtgV-o",
	}
	videos <- app.Video{
		Id:  3,
		Url: "https://www.youtube.com/watch?v=a4na2opArGY",
	}
	videos <- app.Video{
		Id:  4,
		Url: "https://www.youtube.com/watch?v=0YF8vecQWYs",
	}
	videos <- app.Video{
		Id:  5,
		Url: "https://www.youtube.com/watch?v=pmanD_s7G3U",
	}
	videos <- app.Video{
		Id:  6,
		Url: "https://www.youtube.com/watch?v=atxYe-nOa9w",
	}
	videos <- app.Video{
		Id:  7,
		Url: "https://www.youtube.com/watch?v=792vg0amsuQ",
	}
	videos <- app.Video{
		Id:  8,
		Url: "https://www.youtube.com/watch?v=_FDEH7hWb8c",
	}
	videos <- app.Video{
		Id:  9,
		Url: "https://www.youtube.com/watch?v=JdSpuTi9d8A",
	}
	videos <- app.Video{
		Id:  10,
		Url: "https://www.youtube.com/watch?v=EZKzXnq6ppk",
	}
	<-ctx.Done()
	close(videos)
}
