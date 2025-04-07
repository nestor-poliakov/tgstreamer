package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"tgstreamer/internal/app"
	"tgstreamer/internal/rpc"
	"tgstreamer/internal/streamer"
)

func main() {
	stopContext := context.Background()
	stopContext, stopFunc := signal.NotifyContext(stopContext, syscall.SIGINT, syscall.SIGTERM)
	defer stopFunc()

	conf := app.ReadConfig()
	fmt.Printf("%+v\n", conf)

	client := rpc.NewYtDlpClient(conf.VideosDir)
	fmt.Println(client.Download(stopContext, "https://www.youtube.com/watch?v=qjxxYoL7nSU"))

	streamer.PlayVideos(stopContext, conf.StreamingUrl)
}
