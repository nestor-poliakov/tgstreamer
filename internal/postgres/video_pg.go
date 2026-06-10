package postgres

import (
	"context"
	"fmt"
	"strings"

	"tgstreamer/internal/app"

	"tgstreamer/lib/pg"
)

type Video struct{}

func NewVideo() Video {
	return Video{}
}

func (Video) AddFileInfo(ctx context.Context, id int64, fileInfo app.FileInfo) error {
	sql := `update video set file_info = $1 where id = $2`
	return pg.FromContext(ctx).Query(sql, fileInfo, id).ExecContext(ctx)
}

func (Video) AddYoutubeInfos(ctx context.Context, videos []app.Video) error {
	var sql strings.Builder
	vals := make([]any, len(videos)*2)
	for i, v := range videos {
		vals[i*2] = v.YtInfo
		vals[i*2+1] = v.Id
		fmt.Fprintf(&sql, `update video set yt_info = $%d where id = $%d;`, i*2+1, i*2+2)
	}
	return pg.FromContext(ctx).Query(sql.String(), vals...).ExecContext(ctx)
}

func (Video) AddSponsorBlockInfo(ctx context.Context, id int64, info app.SponsorBlockInfo) error {
	sql := `update video set sb_info = $1 where id = $2`
	return pg.FromContext(ctx).Query(sql, info, id).ExecContext(ctx)
}

func (Video) Get(ctx context.Context, id int64) (video app.Video, err error) {
	sql := `select id, code, created_at, file_info, yt_info, sb_info from video where id = $1`
	return video, pg.FromContext(ctx).Query(sql, id).LoadOneContext(ctx, &video)
}

func (Video) GetNoYtInfo(ctx context.Context) (res []app.Video, err error) {
	sql := `select id, code, created_at, file_info, sb_info
			from video
			where yt_info = '{}'
			limit 50`
	return res, pg.FromContext(ctx).Query(sql).LoadContext(ctx, &res)
}

func (Video) GetNoSbInfo(ctx context.Context) (video app.Video, err error) {
	sql := `select id, code, created_at, file_info, yt_info
			from video
			where sb_info = '{}'
			limit 1`
	return video, pg.FromContext(ctx).Query(sql).LoadOneContext(ctx, &video)
}

func (Video) Create(ctx context.Context, video app.Video) (res app.Video, err error) {
	sql := `insert into video (code, yt_info)
	values ($1,$2)
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
		qs[i] = fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2)
	}
	sql := fmt.Sprintf(`insert into video (code, yt_info) values %s
	on conflict (code) do update set yt_info = excluded.yt_info
	returning id, code, created_at`, strings.Join(qs, ","))
	return res, pg.FromContext(ctx).Query(sql, vals...).LoadContext(ctx, &res)
}

func (Video) DeleteFileNames(ctx context.Context, fileNames []string) error {
	vals := make([]any, len(fileNames))
	placeholders := make([]string, len(fileNames))
	for i := range fileNames {
		vals[i] = fileNames[i]
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	sql := `update video
			set (file_info) = ('{}')
			where file_info ->> 'name' is not null
			and file_info ->> 'name' in (` + strings.Join(placeholders, ",") + `)`
	return pg.FromContext(ctx).Query(sql, vals...).ExecContext(ctx)
}
