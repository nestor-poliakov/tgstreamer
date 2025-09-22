# Tg Streamer
Infinite stream to telegram channel with videos from YouTube

Working example: [@anime_openings_stream](https://t.me/anime_openings_stream)

Known Problems:
- When no one watching strem for some time, telegram shut down processing, but still reads packets from connection, and some times it is not starting processing, so there is no stream on client side.
- Change in video resolution, number of audio channels or audio bitrate breaks the stream, so it needs to reconnect.
  Solution: filter videos by resolution and number of channels.
-

- [x] download videos with yt-dlp
- [x] stream one flv file
- [x] stream multiple files one by one
- [x] stream mp4 file
- [x] download videos before streaming
- [x] stream videos from playlist in db
- [x] run multiple streams simultaneously
- [x] download video title and cover
- [x] download segments from SponsorBlock
- [x] delete files after streaming
- [x] send video cover to channel when it starts
- [x] form playlist from youtube playlists
- [x] filter video by resolution
- [x] BUG if 1 video occurs multiple times in a row in playlist, rate limiter not working
- [x] form playlist from youtube channels
- [x] cut ad using data from SponsorBlock
- [ ] predownloading videos before streaming pipeline
- [ ] vote for skipping video
- [ ] BUG ad cutter not working
- [ ] better video quality
- [ ] make video playlist automatically with new videos by list of youtube channels
- [ ] receive suggestions about new videos
- [ ] make list of future videos
- [ ] collect data about number of people watching
