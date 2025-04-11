package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/postgres"
	"tgstreamer/internal/rpc"
	"tgstreamer/internal/streamer"
	"tgstreamer/lib/log"
	"tgstreamer/lib/pg"
)

func main() {
	stopContext := context.Background()
	stopContext, stopFunc := signal.NotifyContext(stopContext, syscall.SIGINT, syscall.SIGTERM)
	defer stopFunc()
	ctx := context.Background()
	conf := app.ReadConfig()
	fmt.Printf("%+v\n", conf)

	pg.Migrate(postgres.Migrations, conf.Pg)
	pgConn := pg.NewConn(stopContext, conf.Pg.Dsn)
	ctx = pg.NewContext(ctx, pgConn)

	var (
		streamPg   = postgres.NewStream()
		videoPg    = postgres.NewVideo()
		playlistPg = postgres.NewPlaylist()
	)
	var (
		youtube      = rpc.NewYoutube(stopContext, conf.YoutubeApiKey)
		sponsorBlock = rpc.NewSponsorBlock()
	)

	var (
		playlistLogic = logic.NewPlaylist(playlistPg, videoPg)
		videoLogic    = logic.NewVideo(videoPg, youtube, sponsorBlock)
	)
	var (
		ytdlp = rpc.NewYtDlpClient(videoLogic, conf.VideosDir)
	)

	manager := streamer.NewManager(playlistLogic, *streamPg, ytdlp)
	fillDb(ctx)
	manager.Run(ctx)
	playlistLogic.Run(ctx)
	videoLogic.Run(ctx)
	<-stopContext.Done()
	log.Default().Info("stop signal received")
	manager.Stop()
	playlistLogic.Stop()
	videoLogic.Stop()
}

func fillDb(ctx context.Context) {
	codes := []string{"4evV8Fr5A8U", "8OkpRK2_gVs", "jIfogFtgV-o", "a4na2opArGY", "0YF8vecQWYs", "pmanD_s7G3U",
		"atxYe-nOa9w", "792vg0amsuQ", "_FDEH7hWb8c", "JdSpuTi9d8A", "EZKzXnq6ppk"}
	for _, code := range codes {
		video, err := postgres.NewVideo().Create(ctx, app.Video{
			Code: code,
		})
		if err != nil {
			panic(err)
		}
		err = postgres.NewPlaylist().Create(ctx, video.Id, 1)
		if err != nil {
			panic(err)
		}
	}
}
