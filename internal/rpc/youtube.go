package rpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tgstreamer/internal/app"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Youtube struct {
	service *youtube.Service
}

func NewYoutube(ctx context.Context, apiKey string) *Youtube {
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(err)
	}
	y := &Youtube{
		service: service,
	}
	return y
}

func (y *Youtube) GetInfo(ctx context.Context, youtubeVideoId string) (app.YoutubeInfo, error) {
	resp, err := y.service.Videos.List([]string{"snippet,contentDetails,statistics"}).Id(youtubeVideoId).Do()
	if err != nil {
		return app.YoutubeInfo{}, fmt.Errorf("get info: %w", err)
	}
	if len(resp.Items) == 0 {
		return app.YoutubeInfo{}, fmt.Errorf("no items in response")
	}
	v := resp.Items[0]
	publishedAt, err := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
	if err != nil {
		return app.YoutubeInfo{}, fmt.Errorf("parse published at: %w", err)
	}
	var thumbnail = &youtube.Thumbnail{}
	for _, tn := range []*youtube.Thumbnail{
		v.Snippet.Thumbnails.Maxres,
		v.Snippet.Thumbnails.High,
		v.Snippet.Thumbnails.Medium,
		v.Snippet.Thumbnails.Standard,
		v.Snippet.Thumbnails.Default,
	} {
		if tn != nil {
			thumbnail = tn
			break
		}
	}
	return app.YoutubeInfo{
		PublishedAt: publishedAt.Unix(),
		Thumbnail:   thumbnail.Url,
		Title:       v.Snippet.Title,
		Duration:    y.parseDuration(v.ContentDetails.Duration),
	}, nil
}

func (y *Youtube) parseDuration(str string) int64 {
	var dur time.Duration
	if len(str) < 3 {
		return 0
	}
	str = str[2:]
	h := strings.Index(str, "H")
	if h != -1 {
		hours, _ := strconv.Atoi(str[:h])
		dur += time.Hour * time.Duration(hours)
		str = str[h+1:]
	}
	m := strings.Index(str, "M")
	if m != -1 {
		mins, _ := strconv.Atoi(str[:m])
		dur += time.Minute * time.Duration(mins)
		str = str[m+1:]
	}
	s := strings.Index(str, "S")
	if s != -1 {
		secs, _ := strconv.Atoi(str[:s])
		dur += time.Second * time.Duration(secs)
	}
	return int64(dur.Seconds())
}
