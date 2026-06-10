-- +goose Up
create table video (
    id            bigserial not null primary key,
    code          text      not null unique,
    created_at    bigint    not null default extract(epoch from now()),
    file_info     jsonb     not null default '{}',
    yt_info       jsonb     not null default '{}',
    sb_info       jsonb     not null default '{}'
);

create table stream (
    id        bigserial not null primary key,
    name      text      not null default '',
    is_active bool      not null default false,
    type      text      not null default null,
    settings  jsonb     not null default '{}'
);

create table playlist_item (
    id         bigserial not null primary key,
    stream_id  bigint    not null references stream(id),
    video_id   bigint    not null references video(id),
    is_current bool      not null default false
);

create unique index unique_playlist_item_stream_id_current
on playlist_item (stream_id)
where is_current = true;


create table play (
    id               bigserial not null primary key,
    created_at       bigint    not null default extract(epoch from now()),
    playlist_item_id bigint    not null references playlist_item(id),
    announce_msg_id  bigint    not null,
    likes            bigint    not null default 0,
    skips            bigint    not null default 0
);
