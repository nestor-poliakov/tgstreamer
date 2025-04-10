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
	query := `update video set (file_name,downloaded_at) = (?,?) where id = ?`
	return pg.FromContext(ctx).Query(query, fileName, time.Now().Unix(), id).ExecContext(ctx)
}

func (Video) AddYoutubeInfo(ctx context.Context, id int64, info app.YoutubeInfo) error {
	query := `update video set yt_info = ? where id = ?`
	return pg.FromContext(ctx).Query(query, info, id).ExecContext(ctx)
}

func (Video) AddSponsorBlockInfo(ctx context.Context, id int64, info app.SponsorBlockInfo) error {
	query := `update video set sb_info = ? where id = ?`
	return pg.FromContext(ctx).Query(query, info, id).ExecContext(ctx)
}

func (Video) Get(ctx context.Context, id int64) (video app.Video, err error) {
	query := `select id, code, created_at, downloaded_at, file_name, yt_info, sb_info from video where id = ?`
	return video, pg.FromContext(ctx).Query(query, id).LoadOneContext(ctx, &video)
}

func (Video) Create(ctx context.Context, video app.Video) (res app.Video, err error) {
	sql := `insert into video(code) values (?) returning id, code, created_at`
	return res, pg.FromContext(ctx).Query(sql, video.Code).LoadOneContext(ctx, &res)
}
