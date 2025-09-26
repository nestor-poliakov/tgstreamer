package app

import (
	"os"
	"strconv"
	"tgstreamer/lib/pg"
)

type Config struct {
	VideosDir     string
	CookiesFile   string
	YoutubeApiKey string
	Pg            pg.Config
	TgBotToken    string
}

func ReadConfig() (conf Config) {
	return Config{
		VideosDir:     os.Getenv("VIDEOS_DIR"),
		CookiesFile:   os.Getenv("COOKIES_FILES"),
		YoutubeApiKey: os.Getenv("YOUTUBE_API_KEY"),
		Pg: pg.Config{
			Dsn:        os.Getenv("PG.DSN"),
			DropTables: envBool("PG.DROP_TABLES"),
		},
		TgBotToken: os.Getenv("TG_BOT_TOKEN"),
	}
}

func envBool(e string) (r bool) {
	r, _ = strconv.ParseBool(os.Getenv(e))
	return r
}

func envInt64(e string) (r int64) {
	r, _ = strconv.ParseInt(os.Getenv(e), 10, 64)
	return r
}
