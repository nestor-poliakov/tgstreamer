package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tgstreamer/internal/rpc"
)

func main() {
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	// https://www.youtube.com/@SomeChannel
	chanLink := os.Getenv("CHAN_LINK")
	u, err := url.Parse(chanLink)
	if err != nil {
		panic(err)
	}
	chanName := strings.ReplaceAll(u.Path, "/", "")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	yt := rpc.NewYoutube(ctx, apiKey)

	channelId, err := yt.SearchChannel(ctx, chanName)
	if err != nil {
		panic(err)
	}
	fmt.Println(chanName + " channel id: " + channelId)
}
