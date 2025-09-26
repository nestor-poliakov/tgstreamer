package postgres

import (
	"context"
	"fmt"
	"strings"

	"tgstreamer/internal/app"
	"tgstreamer/lib/pg"
)

type Playlist struct{}

func NewPlaylist() Playlist {
	return Playlist{}
}

func (Playlist) Create(ctx context.Context, videoId int64, streamId int64) error {
	sql := `insert into playlist_item (video_id, stream_id) values ($1, $2)`
	return pg.FromContext(ctx).Query(sql, videoId, streamId).ExecContext(ctx)
}

func (Playlist) CreateList(ctx context.Context, videoIds []int64, streamId int64) error {
	if len(videoIds) == 0 {
		return nil
	}
	vals := make([]any, len(videoIds)*2)
	qs := make([]string, len(videoIds))
	for i := range videoIds {
		vals[i*2] = videoIds[i]
		vals[i*2+1] = streamId
		qs[i] = "(?,?)"
	}
	sql := `insert into playlist_item (video_id,stream_id) values ` + strings.Join(qs, ",")
	return pg.FromContext(ctx).Query(sql, vals...).ExecContext(ctx)
}

func (Playlist) GetForStream(ctx context.Context, streamId int64) (res []app.PlaylistItem, err error) {
	sql := `select id, video_id, stream_id from playlist_item where stream_id = $1`
	return res, pg.FromContext(ctx).Query(sql, streamId).LoadContext(ctx, &res)
}

func (Playlist) SetCurrent(ctx context.Context, id int64) error {
	ctx, tx, err := pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.RollbackUnlessCommitted()
	sql := `update playlist_item
			set is_current = false
			where stream_id = (select stream_id from playlist_item where id = $1)
			and is_current;`

	err = pg.FromContext(ctx).Query(sql, id).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("clear current: %w", err)
	}

	sql = `update playlist_item
			set is_current = true
			where stream_id = (select stream_id from playlist_item where id = $1)
			and id = $1;`
	err = pg.FromContext(ctx).Query(sql, id).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("set current: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (Playlist) Get(ctx context.Context, id int64) (res app.PlaylistItem, err error) {
	sql := `select id, video_id, stream_id from playlist_item where id = $1`
	return res, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &res)
}

func (Playlist) GetCurrent(ctx context.Context, streamId int64) (res app.PlaylistItem, err error) {
	sql := `select id, video_id, stream_id from playlist_item where stream_id = $1 and is_current = true`
	return res, pg.FromContext(ctx).Query(sql, streamId).LoadOneContext(ctx, &res)
}

func (Playlist) GetNext(ctx context.Context, playlistItemId int64) (res app.PlaylistItem, err error) {
	sql := `select id, video_id, stream_id
			from playlist_item
			where stream_id = (select stream_id FROM playlist_item where id = $1)
			and id > $2
			order by id asc
			limit 1;`
	return res, pg.FromContext(ctx).Query(sql, playlistItemId, playlistItemId).LoadOneContext(ctx, &res)
}

func (Playlist) GetFirst(ctx context.Context, streamId int64) (res app.PlaylistItem, err error) {
	sql := `select id, video_id, stream_id from playlist_item where stream_id = $1 order by id asc limit 1`
	return res, pg.FromContext(ctx).Query(sql, streamId).LoadOneContext(ctx, &res)
}
