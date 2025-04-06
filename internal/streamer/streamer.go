package streamer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"tgstreamer/lib/log"
	"time"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/nestor-poliakov/joy5/format/flv"
	"github.com/nestor-poliakov/joy5/format/flv/flvio"
	_ "github.com/nestor-poliakov/joy5/format/flv/flvio"
	"github.com/nestor-poliakov/joy5/format/rtmp"
)

func Stream(ctx context.Context, fileName, streamUrl string) {
	if !strings.HasSuffix(fileName, ".flv") {
		panic(fmt.Errorf("unsupported file format %q", fileName))
	}
	file, err := os.Open(fileName)
	if err != nil {
		panic(fmt.Errorf("failed to open file %q: %w", fileName, err))
	}
	defer file.Close()
	stream(ctx, file, streamUrl)
}

func stream(ctx context.Context, f io.Reader, streamingUrl string) {
	demuxer := flv.NewDemuxer(f)

	client := rtmp.NewClient()
	client.LogEvent = func(c *rtmp.Conn, nc net.Conn, e int) {
		log.Defaults().Debugf("Event: %s", rtmp.EventString[e])
	}
	conn, nconn, err := client.Dial(streamingUrl, rtmp.PrepareWriting)
	defer nconn.Close()
	if err != nil {
		panic(fmt.Errorf("failed to connect to rtmp server %q: %w", streamingUrl, err))
	}
	defer nconn.Close()
	var rl *rateLimiter
	var t time.Time
	var i int
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		packet, err := demuxer.ReadPacket()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			panic(fmt.Errorf("failed to read packet: %w", err))
		}
		if packet.Type == av.Metadata {
			arr, _ := flvio.ParseAMFVals(packet.Data, false)
			m := arr[0].(flvio.AMFMap)
			m.Set("duration", 0)
			m.Set("filesize", 0)
			data := make([]byte, flvio.FillAMF0Vals(nil, []any{m}))
			flvio.FillAMF0Vals(data, []any{m})
			packet.Data = data
		}
		if packet.Type > 2 {
			rl = newRateLimiter()
			t = time.Now()
		} else {
			rl.Limit(packet)
		}
		if i%1000 == 0 || packet.Type > 2 {
			fmt.Printf("%d write packet %s %s %s\n", i, av.PacketTypeString[packet.Type], packet.Time, time.Since(t)-packet.Time)
		}
		if err := conn.WritePacket(packet); err != nil {
			panic(fmt.Errorf("failed to write packet to rtmp server %q: %w", streamingUrl, err))
		}
		i++
	}
}
