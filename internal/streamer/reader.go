package streamer

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"os"
	"strings"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/logic"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/nestor-poliakov/joy5/format/flv"
	"github.com/nestor-poliakov/joy5/format/flv/flvio"
	goflv "github.com/yapingcat/gomedia/go-flv"
	gomp4 "github.com/yapingcat/gomedia/go-mp4"
)

type Reader struct {
	videos     <-chan app.Video
	ch         chan<- piece
	downloader *rpc.YtDlpClient
	videoLogic *logic.Video
}

func NewReader(videos <-chan app.Video, ch chan<- piece, downloader *rpc.YtDlpClient, videoLogic *logic.Video) *Reader {
	return &Reader{
		videos:     videos,
		ch:         ch,
		downloader: downloader,
		videoLogic: videoLogic,
	}
}

func (r *Reader) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx = log.With(ctx, "worker", "reader")
	wg.Add(1)
	go r.readingLoop(ctx, wg)
}

func (r *Reader) readingLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer log.FromContexts(ctx).Info("reader stopped")
	defer close(r.ch)
	vctx := ctx
	for {
		select {
		case <-ctx.Done():
			return
		case video, ok := <-r.videos:
			if !ok {
				return
			}
			vctx = log.With(ctx, "video_id", video.Id)
			err := r.processVideo(vctx, video)
			if err != nil {
				log.FromContexts(vctx).With("error", err).Errorf("process video %q", video.FileName)
			}
		}
	}
}

func (r *Reader) processVideo(ctx context.Context, video app.Video) (err error) {
	video.FileName, err = r.downloader.DownloadYt(ctx, video.Code)
	if err != nil {
		return fmt.Errorf("download video: %w", err)
	}
	r.videoLogic.SetDownloaded(video.Id, video.FileName)
	log.FromContexts(ctx).Infof("start reading new video %q", video.FileName)
	var reader io.ReadCloser
	var m flvio.AMFMap
	switch video.FileName[len(video.FileName)-3:] {
	case "mp4":
		reader, m, err = r.getMp4Reader(ctx, video.FileName)
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
	return r.processReader(ctx, reader, video, m)
}

func (r *Reader) getMp4Reader(ctx context.Context, fileName string) (io.ReadCloser, flvio.AMFMap, error) {
	pr, pw := io.Pipe()
	f, err := os.Open(fileName)
	if err != nil {
		return nil, nil, fmt.Errorf("open file %q: %w", fileName, err)
	}
	demuxer := gomp4.CreateMp4Demuxer(f)
	tracks, err := demuxer.ReadHead()
	if err != nil {
		return nil, nil, fmt.Errorf("read mp4 header: %w", err)
	}
	muxer := goflv.CreateFlvWriter(bufio.NewWriter(pw))
	err = muxer.WriteFlvHeader()
	if err != nil {
		return nil, nil, fmt.Errorf("write flv header: %w", err)
	}
	m := ConvertToMetadata(demuxer.GetMp4Info(), tracks)
	go func() {
		defer log.FromContext(ctx).Info("stop reading mp4 file")
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
			case gomp4.MP4_CODEC_H264:
				err = muxer.WriteH264(p.Data, uint32(p.Pts), uint32(p.Dts))
			case gomp4.MP4_CODEC_AAC:
				err = muxer.WriteAAC(p.Data, uint32(p.Pts), uint32(p.Dts))
			default:
				err = fmt.Errorf("unknoun packet type %d", p.Cid)
			}
			if errors.Is(err, io.ErrClosedPipe) {
				return
			}
			if err != nil {
				log.FromContexts(ctx).With("error", err).Errorf("write flv packet")
				return
			}
		}
	}()

	return pr, m, nil
}

func (r *Reader) getFlvReader(fileName string) (io.ReadCloser, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", fileName, err)
	}
	return f, nil
}

func (r *Reader) processReader(ctx context.Context, reader io.Reader, video app.Video, m flvio.AMFMap) error {
	demuxer := flv.NewDemuxer(reader)

	err := demuxer.ReadFileHeader()
	if err != nil {
		return fmt.Errorf("read file header: %w", err)
	}
	data := make([]byte, flvio.FillAMF0Vals(nil, []any{m}))
	flvio.FillAMF0Vals(data, []any{m})
	keyFrames := []av.Packet{
		{
			Type: av.Metadata,
			Data: data,
		},
	}
	log.FromContexts(ctx).Infof("start processing file %q", video.FileName)
	for {
		packets, keyFrame, err := r.readUntilKeyFrame(demuxer, keyFrames)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read until key frame: %w", err)
		}
		select {
		case r.ch <- piece{
			videoId: video.Id,
			packets: packets,
		}:
		case <-ctx.Done():
			return ctx.Err()
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

func ConvertToMetadata(info gomp4.Mp4Info, tracks []gomp4.TrackInfo) flvio.AMFMap {
	var (
		width, height              uint32
		videoRate, frameRate       float64
		audioRate                  float64
		audioSampleRate            uint32
		audioSampleSize            uint16
		stereo                     bool
		videoCodecId, audioCodecId int
	)

	for _, t := range tracks {
		switch t.Cid {
		case gomp4.MP4_CODEC_H264:
			width = t.Width
			height = t.Height
			if t.Duration > 0 && t.Timescale > 0 {
				durationSec := float64(t.Duration) / float64(t.Timescale)
				frameRate = float64(t.SampleCount) / durationSec
			}
			videoCodecId = 7 // RTMP codec ID for AVC
		case gomp4.MP4_CODEC_AAC:
			audioSampleRate = t.SampleRate
			audioSampleSize = t.SampleSize
			stereo = t.ChannelCount > 1
			audioCodecId = 10 // RTMP codec ID for AAC
			if t.Duration > 0 && t.Timescale > 0 {
				durationSec := float64(t.Duration) / float64(t.Timescale)
				audioRate = (float64(t.SampleCount) * float64(t.SampleSize) * 8.0) / 1000.0 / durationSec
				audioRate = 0
			}
		}
	}
	compatableBrands := strings.Builder{}
	for _, b := range info.CompatibleBrands {
		compatableBrands.WriteString(FourCCToString(b))
	}
	// Metadata
	metadata := flvio.AMFMap{
		{K: "duration", V: float64(0)},
		{K: "width", V: float64(width)},
		{K: "height", V: float64(height)},
		{K: "videodatarate", V: videoRate},
		{K: "framerate", V: frameRate},
		{K: "videocodecid", V: float64(videoCodecId)},
		{K: "audiodatarate", V: audioRate},
		{K: "audiosamplerate", V: float64(audioSampleRate)},
		{K: "audiosamplesize", V: float64(audioSampleSize)},
		{K: "stereo", V: stereo},
		{K: "audiocodecid", V: float64(audioCodecId)},
		{K: "major_brand", V: FourCCToString(info.MajorBrand)},
		{K: "minor_version", V: float64(info.MinorVersion)},
		{K: "compatible_brands", V: compatableBrands.String()},
		{K: "encoder", V: "Lavf61.7.100"},
	}

	return metadata
}

func FourCCToString(fourcc uint32) string {
	bytes := make([]byte, 4)
	binary.NativeEndian.PutUint32(bytes, fourcc)
	return string(bytes)
}
