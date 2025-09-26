package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	tg   *rpc.Telegram
	play *logic.Play

	wg       sync.WaitGroup
	stopFunc func()
}

func NewHandler(play *logic.Play, tg *rpc.Telegram) *Handler {
	return &Handler{
		play: play,
		tg:   tg,
	}
}

func (h *Handler) Run(ctx context.Context) {
	ctx, h.stopFunc = context.WithCancel(ctx)
	h.wg.Add(1)
	go h.processingLoop(ctx)
}

func (h *Handler) Stop() {
	h.stopFunc()
	h.wg.Wait()
}

func (h *Handler) processingLoop(ctx context.Context) {
	defer h.wg.Done()
	ctx = log.With(ctx, "worker", "tg_handler")
	updatesCh := h.tg.GetUpdatesChan()
	for {
		select {
		case <-ctx.Done():
			return
		case update, ok := <-updatesCh:
			if !ok {
				return
			}
			ctx := log.With(ctx, "update_id", update.UpdateID)
			err := h.processUpdate(ctx, update)
			if err != nil {
				log.FromContext(ctx).With("error", err).Error("failed to process update")
			}
		}
	}
}

func (h *Handler) processUpdate(ctx context.Context, update tgbotapi.Update) (err error) {
	switch {
	case update.CallbackQuery != nil:
		log.FromContexts(ctx).Debugf("processing callback: %v", *update.CallbackQuery)
		err = h.processCallback(ctx, *update.CallbackQuery)
	default:
		err = fmt.Errorf("unknown message %v", update)
	}
	return err
}

func (h *Handler) processCallback(ctx context.Context, callback tgbotapi.CallbackQuery) error {
	h.tg.Callback(ctx, callback.ID, "")
	data := strings.Split(callback.Data, ":")
	if len(data) != 2 {
		return fmt.Errorf("invalid callback data %q", callback.Data)
	}
	id, err := strconv.ParseInt(data[1], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse id %q: %w", data[1], err)
	}
	switch data[0] {
	case "like":
		return h.play.Like(ctx, id)
	case "skip":
		return h.play.Skip(ctx, id)
	default:
		return fmt.Errorf("unknown callback type %q", data[0])
	}
}
