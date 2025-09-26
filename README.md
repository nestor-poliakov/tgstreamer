# TgStreamer

**YouTube-to-Telegram Live Streaming Service**

Continuously stream videos from YouTube playlists and channels directly into your Telegram channels.

Key features:
- Playlist forms from YouTube playlists or channels
- Automatic video downloading via yt-dlp
- Ad removal with SponsorBlock integration
- Support for multiple simultaneous streams
- Automatic announcements for new videos
- User voting system to skip videos

**Live Example**: [@anime_openings_stream](https://t.me/anime_openings_stream)

## Known Problems
- When no one watching stream for some time, telegram shut down processing, but still reads packets from connection. When someone joins the steam, it is not starting processing, so there is no stream on client side. Somtimes waiting for new video or rejoining to stream helps.
- Change in video resolution, number of audio channels or audio bitrate breaks the stream. Current solution: filter videos by resolution, number of channels and audio bitrate.


## Configuration
```VIDEOS_DIR``` - path to directory to store videos

```COOKIES_FILES``` - path to cookies file for youtube

```YOUTUBE_API_KEY``` - api key from gcp

```PG.DSN``` - connection string to postgresql db

```TG_BOT_TOKEN``` - telegram bot token


## Task list

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
- [x] form playlist from youtube channel
- [x] cut ad using data from SponsorBlock
- [x] ~~predownloading videos before streaming pipeline~~
- [x] vote for skipping video
- [ ] BUG ad cutter not working
- [ ] better video quality
- [ ] make video playlist automatically with new videos by list of youtube channels
- [ ] receive suggestions about new videos
- [ ] make list of future videos
- [ ] collect data about number of people watching
