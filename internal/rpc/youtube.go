package rpc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tgstreamer/internal/app"
	"tgstreamer/lib/log"
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

func (y *Youtube) GetInfo(ctx context.Context, youtubeVideoIds []string) (res []app.YoutubeInfo, err error) {
	if len(youtubeVideoIds) > 50 {
		return nil, fmt.Errorf("too many videos to process")
	}
	resp, err := y.service.Videos.List([]string{"snippet,contentDetails,statistics"}).Id(youtubeVideoIds...).Do()
	if err != nil {
		return nil, fmt.Errorf("get info: %w", err)
	}
	resMap := make(map[string]app.YoutubeInfo, len(resp.Items))
	for _, v := range resp.Items {
		publishedAt, _ := time.Parse(time.RFC3339, v.Snippet.PublishedAt)
		resMap[v.Id] = app.YoutubeInfo{
			PublishedAt: publishedAt.Unix(),
			Thumbnail:   y.getThumbnailUrl(*v.Snippet.Thumbnails),
			Title:       v.Snippet.Title,
		}
	}
	for _, id := range youtubeVideoIds {
		info := resMap[id]
		info.LoadedAt = time.Now().Unix()
		res = append(res, info)
	}
	return res, nil
}

func (y *Youtube) GetPlaylist(ctx context.Context, playlistId string) (videos []app.Video, err error) {
	pageToken := "0"
	for pageToken != "" {
		call := y.service.PlaylistItems.List([]string{"snippet,contentDetails"}).
			PlaylistId(playlistId).
			MaxResults(50)
		if pageToken != "" && pageToken != "0" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			log.FromContext(ctx).Error("do request", "error", err)
			break
		}
		pageToken = resp.NextPageToken
		videos = append(videos, y.toVideoCodes(resp.Items)...)
	}
	return videos, nil
}

func (y *Youtube) GetChannelVideos(ctx context.Context, channelId string) (videos []app.Video, err error) {
	resp, err := y.service.Channels.List([]string{"snippet,contentDetails"}).
		Id(channelId).
		Do()

	if err != nil {
		return nil, fmt.Errorf("get channel info: %w", err)
	}
	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("channel %q not found", channelId)
	}
	playlistId := resp.Items[0].ContentDetails.RelatedPlaylists.Uploads
	videos, err = y.GetPlaylist(ctx, playlistId)
	if err != nil {
		return nil, fmt.Errorf("get playlist: %w", err)
	}
	return videos, nil
}

func (y *Youtube) SearchChannel(ctx context.Context, q string) (string, error) {
	resp, err := y.service.Search.List([]string{"snippet"}).
		Q(q).
		Type("channel").
		MaxResults(1).
		Do()

	if err != nil {
		return "", fmt.Errorf("search channel: %w", err)
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("channel %q not found", q)
	}
	return resp.Items[0].Id.ChannelId, nil
}

func (y *Youtube) toVideoCodes(items []*youtube.PlaylistItem) []app.Video {
	res := make([]app.Video, 0, len(items))
	for _, item := range items {
		if item.Snippet.Description == "This video is unavailable." &&
			item.Snippet.Title == "Deleted video" {
			continue
		}
		publishedAt, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		res = append(res, app.Video{
			Code: item.ContentDetails.VideoId,
			YtInfo: app.YoutubeInfo{
				LoadedAt:    time.Now().Unix(),
				PublishedAt: publishedAt.Unix(),
				Thumbnail:   y.getThumbnailUrl(*item.Snippet.Thumbnails),
				Title:       item.Snippet.Title,
			},
		})
	}
	return res
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

func (y *Youtube) getThumbnailUrl(td youtube.ThumbnailDetails) string {
	if td.Maxres != nil {
		return td.Maxres.Url
	}
	if td.High != nil {
		return td.High.Url
	}
	if td.Medium != nil {
		return td.Medium.Url
	}
	if td.Standard != nil {
		return td.Standard.Url
	}
	if td.Default != nil {
		return td.Default.Url
	}
	return ""
}
