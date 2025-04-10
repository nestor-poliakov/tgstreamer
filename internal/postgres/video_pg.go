package postgres

import (
	"context"
	"fmt"
	"time"

	"tgstreamer/internal/app"

	"tgstreamer/lib/pg"
)

type Video struct{}

func NewVideo() Video {
	return Video{}
}

func (Video) GetList(ctx context.Context) ([]app.Video, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (Video) UpdateFileName(ctx context.Context, id int64, fileName string) error {
	sql := `update video set (file_name,downloaded_at) = (?,?) where id = ?`
	return pg.FromContext(ctx).Query(sql, fileName, time.Now().Unix(), id).ExecContext(ctx)
}

func (Video) AddYoutubeInfo(ctx context.Context, id int64, info app.YoutubeInfo) error {
	sql := `update video set yt_info = ? where id = ?`
	return pg.FromContext(ctx).Query(sql, info, id).ExecContext(ctx)
}

func (Video) AddSponsorBlockInfo(ctx context.Context, id int64, info app.SponsorBlockInfo) error {
	sql := `update video set sb_info = ? where id = ?`
	return pg.FromContext(ctx).Query(sql, info, id).ExecContext(ctx)
}

func (Video) Get(ctx context.Context, id int64) (video app.Video, err error) {
	sql := `select id, code, created_at, downloaded_at, file_name, yt_info, sb_info from video where id = ?`
	return video, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &video)
}

func (Video) GetNoYtInfo(ctx context.Context) (video app.Video, err error) {
	sql := `select id, code, created_at, downloaded_at, file_name, sb_info
			from video
			where yt_info = '{}'
			limit 1`
	return video, pg.FromContext(ctx).Query(sql).LoadOneContext(ctx, &video)
}

func (Video) Create(ctx context.Context, video app.Video) (res app.Video, err error) {
	sql := `insert into video(code)
	values (?)
	on conflict (code) do update set code = excluded.code
	returning id, code, created_at`
	return res, pg.FromContext(ctx).Query(sql, video.Code).LoadOneContext(ctx, &res)
}
