package streamer

import (
	"context"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
)

type stream struct {
	stream     app.Stream
	playlist   *Playlist
	downloader *Downloader
	reader     *Reader
	adCutter   *AdCutter
	streamer   *Streamer
	wg         *sync.WaitGroup
	stopFunc   func()
}

func newStream(strm app.Stream, playlist *logic.Playlist, videol *logic.Video, play *logic.Play, downloader *rpc.YtDlpClient) *stream {
	toDownloader := make(chan video, 0)
	toReader := make(chan video, 0)
	toAdCutter := make(chan gop, 0)
	toStreamer := make(chan gop, 20)
	s := &stream{
		stream:   strm,
		playlist: NewPlaylist(toDownloader, strm, playlist),
		downloader: NewDownloader(toDownloader, toReader, downloader, videol,
			strm.Settings.Resolution, strm.Settings.AudioBitrate),
		reader:   NewReader(toReader, toAdCutter),
		adCutter: NewAdCutter(toAdCutter, toStreamer),
		streamer: NewStreamer(toStreamer, strm, playlist, play),
		wg:       &sync.WaitGroup{},
		stopFunc: func() {},
	}
	return s
}

func (s *stream) Run(ctx context.Context) {
	ctx = log.With(ctx, "stream_id", s.stream.Id)
	log.FromContexts(ctx).Infof("start stream with config %+v", s.stream.Settings)
	ctx, s.stopFunc = context.WithCancel(ctx)
	s.playlist.Run(ctx, s.wg)
	s.downloader.Run(ctx, s.wg)
	s.reader.Run(ctx, s.wg)
	s.adCutter.Run(ctx, s.wg)
	s.streamer.Run(ctx, s.wg)
}

func (s *stream) Stop() {
	s.stopFunc()
	s.wg.Wait()
}

func (s *stream) Skip(playlistItemId int64) {
	s.streamer.Skip(playlistItemId)
}
