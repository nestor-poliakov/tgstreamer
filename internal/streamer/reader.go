package streamer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"tgstreamer/internal/app"
	"tgstreamer/lib/log"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/nestor-poliakov/joy5/format/flv"
	goflv "github.com/yapingcat/gomedia/go-flv"
	"github.com/yapingcat/gomedia/go-mp4"
	gomp4 "github.com/yapingcat/gomedia/go-mp4"
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

func (r *Reader) processVideo(ctx context.Context, video app.Video) (err error) {
	log.FromContexts(ctx).Infof("start reading new video %q", video.FileName)
	var reader io.ReadCloser
	switch video.FileName[len(video.FileName)-3:] {
	case "mp4":
		reader, err = r.getMp4Reader(ctx, video.FileName)
	case "flv":
		reader, err = r.getFlvReader(video.FileName)
	default:
		return fmt.Errorf("unknown video format %q", video.FileName)
	}
	if err != nil {
		return err
	}
	defer reader.Close()
	defer log.FromContexts(ctx).Infof("finished reading video %q", video.FileName)
	return r.processReader(ctx, reader, video)
}

// There is a way to read mp4 packets and convert them to flv packets directly.
func (r *Reader) getMp4Reader(ctx context.Context, fileName string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", fileName, err)
	}
	demuxer := gomp4.CreateMp4Demuxer(f)
	_, err = demuxer.ReadHead()
	if err != nil {
		return nil, fmt.Errorf("read mp4 header: %w", err)
	}
	muxer := goflv.CreateFlvWriter(bufio.NewWriter(pw))
	err = muxer.WriteFlvHeader()
	if err != nil {
		return nil, fmt.Errorf("write flv header: %w", err)
	}
	go func() {
		defer f.Close()
		defer pr.CloseWithError(io.EOF)
		for {
			p, err := demuxer.ReadPacket()
			if errors.Is(err, io.EOF) {
				log.FromContext(ctx).Info("end reading mp4 file")
				return
			}
			if err != nil {
				log.FromContexts(ctx).Errorf("read mp4 packet: %w", err)
				return
			}
			switch p.Cid {
			case mp4.MP4_CODEC_H264:
				err = muxer.WriteH264(p.Data, uint32(p.Pts), uint32(p.Dts))
			case mp4.MP4_CODEC_AAC:
				err = muxer.WriteAAC(p.Data, uint32(p.Pts), uint32(p.Dts))
			case mp4.MP4_CODEC_H265:
				err = muxer.WriteH265(p.Data, uint32(p.Pts), uint32(p.Dts))
			default:
				err = fmt.Errorf("unknoun packet type %d", p.Cid)
			}
			if err != nil {
				log.FromContexts(ctx).Errorf("write flv packet: %w", err)
				return
			}
		}
	}()

	return pr, nil
}

func (r *Reader) getFlvReader(fileName string) (io.ReadCloser, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", fileName, err)
	}
	return f, nil
}

func (r *Reader) processReader(ctx context.Context, reader io.Reader, video app.Video) error {
	demuxer := flv.NewDemuxer(reader)
	err := demuxer.ReadFileHeader()
	if err != nil {
		return fmt.Errorf("read file header: %w", err)
	}
	keyFrames := []av.Packet{}
	log.FromContexts(ctx).Infof("start processing file %q", video.FileName)
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
