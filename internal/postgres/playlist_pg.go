package postgres

import (
	"context"

	"tgstreamer/lib/pg"
)

type Playlist struct{}

func NewPlaylist() Playlist {
	return Playlist{}
}

func (Playlist) Create(ctx context.Context, videoId int64, streamId int64) error {
	sql := `insert into playlist_item(video_id, stream_id) values (?, ?)`
	return pg.FromContext(ctx).Query(sql, videoId, streamId).ExecContext(ctx)
}

func (Playlist) SetCurrent(ctx context.Context, id int64) error {
	sql := `begin;

			update playlist_item
			set is_current = false
			where stream_id = (select stream_id from playlist_item where id = ?)
			and is_current;

			update playlist_item
			set is_current = true
			where stream_id = (select stream_id from playlist_item where id = ?)
			and id = ?;

			commit;`
	return pg.FromContext(ctx).Query(sql, id, id, id).ExecContext(ctx)
}

func (Playlist) GetStreamId(ctx context.Context, pliId int64) (streamId int64, err error) {
	sql := `select stream_id from playlist_item where id = ?`
	return streamId, pg.FromContext(ctx).Query(sql, pliId).LoadOneContext(ctx, &streamId)
}

func (Playlist) GetCurrent(ctx context.Context, streamId int64) (pliId int64, videoId int64, err error) {
	p := playlistItem{}
	sql := `select id, video_id from playlist_item where stream_id = ? and is_current = true`
	return p.Id, p.VideoId, pg.FromContext(ctx).Query(sql, streamId).LoadOneContext(ctx, &p)
}

func (Playlist) GetNext(ctx context.Context, playlistItemId int64) (pliId int64, videoId int64, err error) {
	p := playlistItem{}
	sql := `select id, video_id
			from playlist_item
			where stream_id = (select stream_id FROM playlist_item where id = ?)
			and id > ?
			order by id asc
			limit 1;`
	return p.Id, p.VideoId, pg.FromContext(ctx).Query(sql, playlistItemId, playlistItemId).LoadOneContext(ctx, &p)
}

func (Playlist) GetFirst(ctx context.Context, streamId int64) (pliId int64, videoId int64, err error) {
	p := playlistItem{}
	sql := `select id, video_id from playlist_item where stream_id = ? order by id asc limit 1`
	return p.Id, p.VideoId, pg.FromContext(ctx).Query(sql, streamId).LoadOneContext(ctx, &p)
}

type playlistItem struct {
	Id      int64 `db:"id"`
	VideoId int64 `db:"video_id"`
}
