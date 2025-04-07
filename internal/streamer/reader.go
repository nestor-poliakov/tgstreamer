package streamer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"tgstreamer/internal/app"
	"tgstreamer/lib/log"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/nestor-poliakov/joy5/format/flv"
)

type Reader struct {
	videos <-chan app.Video
	ch     chan<- piece
}

func NewReader(videos <-chan app.Video, ch chan<- piece) *Reader {
	return &Reader{
		videos: videos,
		ch:     ch,
	}
}

func (r *Reader) Run(ctx context.Context) {
	ctx = log.With(ctx, "worker", "reader")
	go r.readingLoop(ctx)
}

func (r *Reader) readingLoop(ctx context.Context) {
	defer func() {
		log.FromContexts(ctx).Info("reader stopped")
		close(r.ch)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case video, ok := <-r.videos:
			if !ok {
				return
			}
			err := r.processVideo(ctx, video)
			if err != nil {
				log.FromContexts(ctx).With("error", err).Errorf("process video %q", video.FileName)
			}
		}
	}
}

func (r *Reader) processVideo(ctx context.Context, video app.Video) error {
	log.FromContexts(ctx).Infof("start reading new video %q", video.FileName)
	f, err := os.Open(video.FileName)
	if err != nil {
		return fmt.Errorf("open file %q: %w", video.FileName, err)
	}
	defer log.FromContexts(ctx).Infof("finished reading video %q", video.FileName)
	defer f.Close()
	demuxer := flv.NewDemuxer(f)
	err = demuxer.ReadFileHeader()
	if err != nil {
		return fmt.Errorf("read file header: %w", err)
	}
	keyFrames := []av.Packet{}
	for {
		packets, keyFrame, err := r.readUntilKeyFrame(demuxer, keyFrames)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read until key frame: %w", err)
		}
		r.ch <- piece{
			videoId: video.Id,
			packets: packets,
		}
		keyFrames = []av.Packet{keyFrame}
	}
}

func (r *Reader) readUntilKeyFrame(d *flv.Demuxer, packets []av.Packet) ([]av.Packet, av.Packet, error) {
	for {
		packet, err := d.ReadPacket()
		if err != nil {
			return nil, av.Packet{}, fmt.Errorf("read packet: %w", err)
		}
		if packet.IsKeyFrame {
			return packets, packet, nil
		}
		packets = append(packets, packet)
	}
}
