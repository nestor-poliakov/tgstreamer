package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"tgstreamer/internal/app"
	"tgstreamer/internal/rpc"
	"tgstreamer/internal/streamer"
	"time"
)

func main() {
	stopContext := context.Background()
	stopContext, stopFunc := signal.NotifyContext(stopContext, syscall.SIGINT, syscall.SIGTERM)
	defer stopFunc()
	go func() {
		<-stopContext.Done()
		time.Sleep(time.Second * 10)
		os.Exit(0)
	}()

	conf := app.ReadConfig()
	fmt.Printf("%+v\n", conf)

	client := rpc.NewYtDlpClient(conf.VideosDir)
	fmt.Println(client.Download(stopContext, "https://www.youtube.com/watch?v=qjxxYoL7nSU"))

	streamer.PlayVideos(stopContext, conf)
}
