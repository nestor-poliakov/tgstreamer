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
		ytdlp        = rpc.NewYtDlpClient(conf.VideosDir, conf.CookiesFile)
		youtube      = rpc.NewYoutube(stopContext, conf.YoutubeApiKey)
		sponsorBlock = rpc.NewSponsorBlock()
		tg           = rpc.NewTelegram(conf.TgBotToken)
	)

	var (
		streamPg   = postgres.NewStream()
		videoPg    = postgres.NewVideo()
		playlistPg = postgres.NewPlaylist()
	)

	var (
		playlistLogic = logic.NewPlaylist(playlistPg, videoPg, streamPg, tg)
		videoLogic    = logic.NewVideo(videoPg, youtube, ytdlp, sponsorBlock)
		streamLogic   = logic.NewStream(streamPg, videoPg, playlistPg, youtube)
	)

	manager := streamer.NewManager(playlistLogic, videoLogic, streamLogic, ytdlp)

	manager.Run(ctx)
	playlistLogic.Run(ctx)
	videoLogic.Run(ctx)
	streamLogic.Run(ctx)

	<-stopContext.Done()
	log.Default().Info("stop signal received")
	manager.Stop()
	playlistLogic.Stop()
	videoLogic.Stop()
	streamLogic.Stop()
}
