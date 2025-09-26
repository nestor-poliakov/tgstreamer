package app

type Play struct {
	Id             int64
	CreatedAt      int64
	PlaylistItemId int64
	AnnounceMsgId  int64
	Likes          int64
	Skips          int64
}
