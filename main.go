package main

import (
	"context"
	"fmt"
	"os/signal"
	"tgstreamer/internal/app"
	"tgstreamer/internal/rpc"
)

func main() {
	stopContext := context.Background()
	stopContext, stopFunc := signal.NotifyContext(stopContext)
	defer stopFunc()

	conf := app.ReadConfig()

	client := rpc.NewYtDlpClient("local/videos", conf.YtDlpCmd)
	fmt.Println(client.Download(stopContext, "dQw4w9WgXcQ"))
}
