package postgres

import (
	"context"
	"tgstreamer/internal/app"
	"tgstreamer/lib/pg"
)

type Play struct{}

func NewPlay() Play {
	return Play{}
}

func (p Play) Get(ctx context.Context, id int64) (res app.Play, err error) {
	sql := `select * from play where id = ?`
	return res, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &res)
}

func (p Play) Create(ctx context.Context, play app.Play) (res app.Play, err error) {
	sql := `insert into play (playlist_item_id, announce_msg_id) values (?, ?) returning *`
	return res, pg.FromContext(ctx).Query(sql, play.PlaylistItemId, play.AnnounceMsgId).LoadOneContext(ctx, &res)
}

func (p Play) AddMsgId(ctx context.Context, id int64, msgId int64) error {
	sql := `update play set announce_msg_id = ? where id = ?`
	return pg.FromContext(ctx).Query(sql, msgId, id).ExecContext(ctx)
}

func (p Play) IncLike(ctx context.Context, id int64) (likes int64, err error) {
	sql := `update play set likes = likes + 1 where id = ? returning likes`
	return likes, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &likes)
}

func (p Play) IncSkip(ctx context.Context, id int64) (res app.Play, err error) {
	sql := `update play set skips = skips + 1 where id = ? returning *`
	return res, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &res)
}
