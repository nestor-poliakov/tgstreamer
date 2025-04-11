package rpc

import (
	"context"
	"fmt"
	"tgstreamer/internal/app"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	client *tgbotapi.BotAPI
}

func NewTelegram(token string) *Telegram {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	return &Telegram{
		client: bot,
	}
}

func (c *Telegram) Announce(ctx context.Context, chatId int64, video app.Video) error {
	_, err := c.client.Send(tgbotapi.PhotoConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat: tgbotapi.BaseChat{
				ChatID: chatId,
			},
			File: tgbotapi.FileURL(video.YtInfo.Thumbnail),
		},
		Thumb:           nil,
		Caption:         fmt.Sprintf(`<a href="https://youtube.com/watch?v=%s">%s</a>`, video.Code, video.YtInfo.Title),
		ParseMode:       "html",
		CaptionEntities: nil,
	})
	return err
}
