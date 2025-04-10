package postgres

import (
	"context"

	"tgstreamer/internal/app"
	"tgstreamer/lib/pg"
)

type Stream struct{}

func NewStream() *Stream {
	return &Stream{}
}

func (p Stream) GetActive(ctx context.Context) (res []app.Stream, err error) {
	query := `select id, name, is_active, settings from stream where is_active = true`
	return res, pg.FromContext(ctx).Query(query).LoadContext(ctx, &res)
}
