package app

type Play struct {
	Id             int64 `db:"id"`
	CreatedAt      int64 `db:"created_at"`
	PlaylistItemId int64 `db:"playlist_item_id"`
	AnnounceMsgId  int64 `db:"announce_msg_id"`
	Likes          int64 `db:"likes"`
	Skips          int64 `db:"skips"`
}
