package postgres

import (
	"context"

	"tgstreamer/internal/app"
	"tgstreamer/lib/pg"
)

type Stream struct{}

func NewStream() Stream {
	return Stream{}
}

func (Stream) GetActive(ctx context.Context) (res []app.Stream, err error) {
	query := `select id, name, is_active, type, settings from stream where is_active = true`
	return res, pg.FromContext(ctx).Query(query).LoadContext(ctx, &res)
}

func (Stream) Get(ctx context.Context, id int64) (res app.Stream, err error) {
	query := `select id, name, is_active, type, settings from stream where id = $1`
	return res, pg.FromContext(ctx).Query(query, id).LoadOneContext(ctx, &res)
}
