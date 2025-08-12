package streamer

import (
	"context"
	"sort"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"

	"github.com/nestor-poliakov/joy5/av"
)

type AdCutter struct {
	sb         *rpc.SponsorBlockClient
	in         chan piece
	out        chan piece
	curVideoId int64
	segments   [][2]float64
}

func NewAdCutter(in chan piece, out chan piece) *AdCutter {
	return &AdCutter{
		in:  in,
		out: out,
	}
}

func (c *AdCutter) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx = log.With(ctx, "worker", "ad_cutter")
	wg.Add(1)
	go c.processingLoop(ctx, wg)
}

func (c *AdCutter) processingLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(c.out)
	for {
		select {
		case piece, ok := <-c.in:
			if !ok {
				return
			}
			p := c.processPiece(piece)
			select {
			case <-ctx.Done():
				return
			case c.out <- p:
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *AdCutter) processPiece(piece piece) piece {
	return piece
	if piece.videoId != c.curVideoId {
		c.segments = c.normalizeSegments(piece.Video.SbInfo.Segments, float64(piece.Video.FileInfo.DurationV))
		c.curVideoId = piece.videoId
	}
	if len(c.segments) == 0 {
		return piece
	}
	if c.skipPackets(piece.packets) {
		configPackets := make([]av.Packet, 0, 3)
		for _, packet := range piece.packets {
			if packet.Type > 2 {
				configPackets = append(configPackets, packet)
			}
		}
		piece.packets = configPackets
	}
	return piece
}

func (c *AdCutter) skipPackets(packets []av.Packet) bool {
	if len(c.segments) == 0 {
		return false
	}
	firstNotConfig := av.Packet{}
	for _, packet := range packets {
		if packet.Type <= 2 {
			firstNotConfig = packet
			break
		}
	}
	lastNotConfig := av.Packet{}
	for i := len(packets) - 1; i >= 0; i-- {
		if packets[i].Type <= 2 {
			lastNotConfig = packets[i]
			break
		}
	}

	start := firstNotConfig.Time.Seconds()
	end := lastNotConfig.Time.Seconds()
	if start >= c.segments[0][1] {
		c.segments = c.segments[1:]
	}
	if len(c.segments) == 0 {
		return false
	}
	if start <= c.segments[0][1] || end >= c.segments[0][0] {
		return true
	}
	return false
}

func (c *AdCutter) normalizeSegments(segments []app.Segment, duration float64) [][2]float64 {
	if len(segments) == 0 {
		return nil
	}

	intervals := make([][2]float64, 0, len(segments))
	for _, s := range segments {
		intervals = append(intervals, s.Segment)
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })
	if intervals[0][0] < 3 {
		intervals[0][0] = 0
	}
	if duration-intervals[len(intervals)-1][1] < 3 {
		intervals[len(intervals)-1][1] = duration
	}
	if len(intervals) == 1 {
		return intervals
	}

	result := [][2]float64{intervals[0]}

	for i := 1; i < len(intervals); i++ {
		current := intervals[i]
		last := &result[len(result)-1]

		// Check if current segment overlaps or is close enough to the last segment in result
		// Overlap condition: current[0] <= last[1]
		// Close enough condition: current[0] - last[1] < 3
		if current[0] <= last[1] || current[0]-last[1] < 3 {
			if current[1] > last[1] {
				last[1] = current[1]
			}
		} else {
			result = append(result, current)
		}
	}

	return result
}
