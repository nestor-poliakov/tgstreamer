package postgres

import (
	"context"
	"fmt"
	"strings"
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

func (Video) AddYoutubeInfos(ctx context.Context, videos []app.Video) error {
	sql := ""
	vals := make([]any, len(videos)*2)
	for i, v := range videos {
		vals[i*2] = v.YtInfo
		vals[i*2+1] = v.Id
		sql += `update video set yt_info = ? where id = ?;`
	}
	return pg.FromContext(ctx).Query(sql, vals...).ExecContext(ctx)
}

func (Video) AddSponsorBlockInfo(ctx context.Context, id int64, info app.SponsorBlockInfo) error {
	sql := `update video set sb_info = ? where id = ?`
	return pg.FromContext(ctx).Query(sql, info, id).ExecContext(ctx)
}

func (Video) Get(ctx context.Context, id int64) (video app.Video, err error) {
	sql := `select id, code, created_at, downloaded_at, file_name, yt_info, sb_info from video where id = ?`
	return video, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &video)
}

func (Video) GetNoYtInfo(ctx context.Context) (res []app.Video, err error) {
	sql := `select id, code, created_at, downloaded_at, file_name, sb_info
			from video
			where yt_info = '{}'
			limit 50`
	return res, pg.FromContext(ctx).Query(sql).LoadContext(ctx, &res)
}

func (Video) GetNoSbInfo(ctx context.Context) (video app.Video, err error) {
	sql := `select id, code, created_at, downloaded_at, file_name, yt_info
			from video
			where sb_info = '{}'
			limit 1`
	return video, pg.FromContext(ctx).Query(sql).LoadOneContext(ctx, &video)
}

func (Video) Create(ctx context.Context, video app.Video) (res app.Video, err error) {
	sql := `insert into video (code, yt_info)
	values (?,?)
	on conflict (code) do update set yt_info = excluded.yt_info
	returning id, code, created_at`
	return res, pg.FromContext(ctx).Query(sql, video.Code, video.YtInfo).LoadOneContext(ctx, &res)
}

func (Video) CreateList(ctx context.Context, videos []app.Video) (res []app.Video, err error) {
	vals := make([]any, len(videos)*2)
	qs := make([]string, len(videos))
	for i := range videos {
		vals[i*2] = videos[i].Code
		vals[i*2+1] = videos[i].YtInfo
		qs[i] = "(?,?)"
	}
	sql := fmt.Sprintf(`insert into video (code, yt_info) values %s
	on conflict (code) do update set yt_info = excluded.yt_info
	returning id, code, created_at`, strings.Join(qs, ","))
	return res, pg.FromContext(ctx).Query(sql, vals...).LoadContext(ctx, &res)
}

func (Video) DeleteFileNames(ctx context.Context, fileNames []string) error {
	vals := make([]any, len(fileNames))
	for i := range fileNames {
		vals[i] = fileNames[i]
	}
	sql := `update video set (file_name,downloaded_at) = ('',0) where file_name in (` + questions(len(fileNames)) + `)`
	return pg.FromContext(ctx).Query(sql, vals...).ExecContext(ctx)
}

func questions(n int) string {
	if n < 1 {
		return ""
	}
	var builder strings.Builder
	builder.WriteByte('?')
	for i := 0; i < n-1; i++ {
		builder.WriteString(",?")
	}
	return builder.String()
}
