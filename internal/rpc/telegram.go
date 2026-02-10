package rpc

import (
	"context"
	"fmt"
	"strconv"
	"tgstreamer/internal/app"
	"tgstreamer/lib/log"

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
	log.Defaults().Infof("new tg bot client @%s", bot.Self.UserName)
	return &Telegram{
		client: bot,
	}
}

func (t *Telegram) GetUpdatesChan() tgbotapi.UpdatesChannel {
	return t.client.GetUpdatesChan(tgbotapi.UpdateConfig{})
}

func (c *Telegram) Announce(ctx context.Context, settings app.Settings, video app.Video, playId int64) (int64, error) {
	buttons := []tgbotapi.InlineKeyboardButton{}
	if settings.WithLikeButton {
		like := "like:" + strconv.FormatInt(playId, 10)
		buttons = append(buttons, tgbotapi.InlineKeyboardButton{
			Text:         "Like",
			CallbackData: &like,
		})
	}
	if settings.WithSkipButton {
		skip := "skip:" + strconv.FormatInt(playId, 10)
		buttons = append(buttons, tgbotapi.InlineKeyboardButton{
			Text:         "Skip",
			CallbackData: &skip,
		})
	}
	var markup any
	if len(buttons) > 0 {
		markup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				buttons,
			},
		}
	}

	thumbnail := video.YtInfo.Thumbnail
	if thumbnail == "" {
		thumbnail = fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", video.Code)
	}

	msg, err := c.client.Send(tgbotapi.PhotoConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat: tgbotapi.BaseChat{
				ChatID:      settings.TgChannelId,
				ReplyMarkup: markup,
			},
			File: tgbotapi.FileURL(thumbnail),
		},
		Thumb:           nil,
		Caption:         fmt.Sprintf(`<a href="https://youtube.com/watch?v=%s">%s</a>`, video.Code, video.YtInfo.Title),
		ParseMode:       "html",
		CaptionEntities: nil,
	})
	if err != nil {
		return 0, fmt.Errorf("send announce message to channel: %w", err)
	}
	return int64(msg.MessageID), nil
}

func (c *Telegram) Callback(ctx context.Context, callbackId, text string) error {
	_, err := c.client.Request(tgbotapi.NewCallback(callbackId, text))
	return err
}
