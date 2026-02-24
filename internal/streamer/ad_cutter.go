package streamer

import (
	"context"
	"sort"
	"sync"
	"tgstreamer/internal/app"
	"tgstreamer/lib/log"
	"time"

	"github.com/nestor-poliakov/joy5/av"
)

type AdCutter struct {
	in          chan gop
	out         chan gop
	curVideoId  int64
	timeOffset  time.Duration
	segments    [][2]float64
	segmentIdx  int
}

func NewAdCutter(in chan gop, out chan gop) *AdCutter {
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
		case g, ok := <-c.in:
			if !ok {
				return
			}
			p := c.processGop(g)
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

func (c *AdCutter) processGop(g gop) gop {
	if g.video.Id != c.curVideoId {
		c.segments = c.normalizeSegments(g.video.SbInfo.Segments, float64(g.video.FileInfo.DurationV))
		c.curVideoId = g.video.Id
		c.timeOffset = 0
		c.segmentIdx = 0
	}

	if len(c.segments) == 0 {
		c.rewritePacketsTime(g.packets)
		return g
	}
	if c.skipPackets(g.packets) {
		c.timeOffset += c.calcDuration(g.packets)
		configPackets := make([]av.Packet, 0)
		for _, packet := range g.packets {
			if packet.Type > 2 {
				configPackets = append(configPackets, packet)
			}
		}
		g.packets = configPackets
	} else {
		c.rewritePacketsTime(g.packets)
	}
	return g
}

func (c *AdCutter) rewritePacketsTime(packets []av.Packet) {
	if c.timeOffset == 0 {
		return
	}
	for i := range packets {
		if packets[i].Type == av.H264 || packets[i].Type == av.AAC {
			packets[i].Time -= c.timeOffset
		}
	}
}

func (c *AdCutter) calcDuration(packets []av.Packet) time.Duration {
	var first, last time.Duration
	found := false
	for _, p := range packets {
		if p.Type == av.H264 || p.Type == av.AAC {
			if !found {
				first = p.Time
				found = true
			}
			last = p.Time
		}
	}
	if !found {
		return 0
	}
	return last - first
}

func (c *AdCutter) skipPackets(packets []av.Packet) bool {
	if c.segmentIdx >= len(c.segments) {
		return false
	}
	firstNotConfig := av.Packet{}
	for _, packet := range packets {
		if packet.Type <= 2 {
			firstNotConfig = packet
			break
		}
	}
	if firstNotConfig.Type == 0 {
		return false
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
	for c.segmentIdx < len(c.segments) && start >= c.segments[c.segmentIdx][1] {
		c.segmentIdx++
	}
	if c.segmentIdx >= len(c.segments) {
		return false
	}
	pieceLen := end - start
	if pieceLen <= 0 {
		return start >= c.segments[c.segmentIdx][0]
	}
	totalOverlap := 0.0
	for i := c.segmentIdx; i < len(c.segments); i++ {
		seg := c.segments[i]
		if seg[0] >= end {
			break
		}
		overlapStart := start
		if seg[0] > overlapStart {
			overlapStart = seg[0]
		}
		overlapEnd := end
		if seg[1] < overlapEnd {
			overlapEnd = seg[1]
		}
		totalOverlap += overlapEnd - overlapStart
	}
	return totalOverlap/pieceLen > 0.5
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
