package streamer

import (
	"fmt"
	"testing"
	"tgstreamer/internal/app"
	"time"

	"github.com/nestor-poliakov/joy5/av"
)

func TestAdCutter_normalizeSegments(t *testing.T) {
	adCutter := &AdCutter{}

	tests := []struct {
		name     string
		segments []app.Segment
		duration float64
		expected [][2]float64
	}{
		{
			name:     "empty segments",
			segments: []app.Segment{},
			duration: 100.0,
			expected: nil,
		},
		{
			name: "single segment",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 20.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 20.0}},
		},
		{
			name: "already sorted segments",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 15.0}},
				{Segment: [2]float64{20.0, 25.0}},
				{Segment: [2]float64{30.0, 35.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 15.0}, {20.0, 25.0}, {30.0, 35.0}},
		},
		{
			name: "unsorted segments need sorting",
			segments: []app.Segment{
				{Segment: [2]float64{30.0, 35.0}},
				{Segment: [2]float64{10.0, 15.0}},
				{Segment: [2]float64{20.0, 25.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 15.0}, {20.0, 25.0}, {30.0, 35.0}},
		},
		{
			name: "overlapping segments should merge",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 20.0}},
				{Segment: [2]float64{15.0, 25.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 25.0}},
		},
		{
			name: "segments with gap less than 3 should merge",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 15.0}},
				{Segment: [2]float64{17.0, 25.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 25.0}},
		},
		{
			name: "segments with gap greater than 3 should not merge",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 15.0}},
				{Segment: [2]float64{19.0, 25.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 15.0}, {19.0, 25.0}},
		},
		{
			name: "first segment close to start (less than 3) should start at 0",
			segments: []app.Segment{
				{Segment: [2]float64{2.0, 10.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{0.0, 10.0}},
		},
		{
			name: "last segment close to end (less than 3) should end at duration",
			segments: []app.Segment{
				{Segment: [2]float64{90.0, 98.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{90.0, 100.0}},
		},
		{
			name: "complex case: sorting, merging, and edge adjustments",
			segments: []app.Segment{
				{Segment: [2]float64{95.0, 98.5}}, // close to end
				{Segment: [2]float64{1.5, 8.0}},   // close to start
				{Segment: [2]float64{20.0, 25.0}},
				{Segment: [2]float64{10.0, 22.0}}, // overlaps with 20-25
				{Segment: [2]float64{30.0, 35.0}},
				{Segment: [2]float64{37.0, 40.0}}, // close to 30-35 (gap = 2)
			},
			duration: 100.0,
			expected: [][2]float64{{0.0, 25.0}, {30.0, 40.0}, {95.0, 100.0}},
		},
		{
			name: "multiple overlapping segments should merge into one",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 20.0}},
				{Segment: [2]float64{15.0, 25.0}},
				{Segment: [2]float64{22.0, 30.0}},
				{Segment: [2]float64{28.0, 35.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 35.0}},
		},
		{
			name: "segments touching exactly should merge",
			segments: []app.Segment{
				{Segment: [2]float64{10.0, 20.0}},
				{Segment: [2]float64{20.0, 30.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{10.0, 30.0}},
		},
		{
			name: "one segment covering entire duration",
			segments: []app.Segment{
				{Segment: [2]float64{1.0, 99.0}},
			},
			duration: 100.0,
			expected: [][2]float64{{0.0, 100.0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adCutter.normalizeSegments(tt.segments, tt.duration)

			if len(result) != len(tt.expected) {
				t.Errorf("normalizeSegments() returned %d segments, expected %d", len(result), len(tt.expected))
				return
			}

			for i, segment := range result {
				if segment[0] != tt.expected[i][0] || segment[1] != tt.expected[i][1] {
					t.Errorf("normalizeSegments() segment %d = [%.1f, %.1f], expected [%.1f, %.1f]",
						i, segment[0], segment[1], tt.expected[i][0], tt.expected[i][1])
				}
			}
		})
	}
}

func TestAdCutter_skipPackets(t *testing.T) {
	tests := []struct {
		name     string
		segments [][2]float64
		packets  []av.Packet
		expected bool
	}{
		{
			name:     "no segments should not skip",
			segments: [][2]float64{},
			packets: []av.Packet{
				{Type: av.H264, Time: 10 * time.Second}, // video packet
				{Type: av.AAC, Time: 11 * time.Second},  // audio packet
			},
			expected: false,
		},
		{
			name:     "no non-config packets should not skip",
			segments: [][2]float64{{5.0, 15.0}},
			packets: []av.Packet{
				{Type: av.H264DecoderConfig, Time: 10 * time.Second}, // config packet
				{Type: av.AACDecoderConfig, Time: 11 * time.Second},  // config packet
			},
			expected: false,
		},
		{
			name:     "packets before segment should not skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 5 * time.Second},
				{Type: av.AAC, Time: 8 * time.Second},
			},
			expected: false,
		},
		{
			name:     "packets after segment should not skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 25 * time.Second},
				{Type: av.AAC, Time: 30 * time.Second},
			},
			expected: false,
		},
		{
			name:     "packets completely within segment should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 12 * time.Second},
				{Type: av.AAC, Time: 15 * time.Second},
			},
			expected: true,
		},
		{
			name:     "minority overlap at start should not skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 6 * time.Second},
				{Type: av.AAC, Time: 12 * time.Second},
			},
			expected: false, // overlap=2s/6s=33%
		},
		{
			name:     "majority overlap at start should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 9 * time.Second},
				{Type: av.AAC, Time: 12 * time.Second},
			},
			expected: true, // overlap=2s/3s=67%
		},
		{
			name:     "minority overlap at end should not skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 18 * time.Second},
				{Type: av.AAC, Time: 28 * time.Second},
			},
			expected: false, // overlap=2s/10s=20%
		},
		{
			name:     "majority overlap at end should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 18 * time.Second},
				{Type: av.AAC, Time: 21 * time.Second},
			},
			expected: true, // overlap=2s/3s=67%
		},
		{
			name:     "packets spanning segment with minority overlap should not skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 5 * time.Second},
				{Type: av.AAC, Time: 30 * time.Second},
			},
			expected: false, // overlap=10s/25s=40%
		},
		{
			name:     "packets mostly within segment should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 9 * time.Second},
				{Type: av.AAC, Time: 21 * time.Second},
			},
			expected: true, // overlap=10s/12s=83%
		},
		{
			name:     "packets touching segment start boundary from inside should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 10 * time.Second},
				{Type: av.AAC, Time: 12 * time.Second},
			},
			expected: true,
		},
		{
			name:     "packets touching segment end boundary from inside should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 18 * time.Second},
				{Type: av.AAC, Time: 20 * time.Second},
			},
			expected: true,
		},
		{
			name:     "mixed config and non-config packets overlapping should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264DecoderConfig, Time: 5 * time.Second},  // config - ignored
				{Type: av.H264, Time: 12 * time.Second},              // video - first non-config
				{Type: av.H264DecoderConfig, Time: 13 * time.Second}, // config - ignored
				{Type: av.AAC, Time: 15 * time.Second},               // audio - last non-config
				{Type: av.AACDecoderConfig, Time: 25 * time.Second},  // config - ignored
			},
			expected: true,
		},
		{
			name:     "multiple segments - packets overlap with first",
			segments: [][2]float64{{10.0, 20.0}, {30.0, 40.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 12 * time.Second},
				{Type: av.AAC, Time: 15 * time.Second},
			},
			expected: true,
		},
		{
			name:     "multiple segments - packets overlap with second",
			segments: [][2]float64{{10.0, 20.0}, {30.0, 40.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 32 * time.Second},
				{Type: av.AAC, Time: 35 * time.Second},
			},
			expected: true,
		},
		{
			name:     "multiple segments - packets between segments should not skip",
			segments: [][2]float64{{10.0, 20.0}, {30.0, 40.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 22 * time.Second},
				{Type: av.AAC, Time: 28 * time.Second},
			},
			expected: false,
		},
		{
			name:     "packets spanning multiple segments with minority overlap should not skip",
			segments: [][2]float64{{10.0, 20.0}, {30.0, 40.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 5 * time.Second},
				{Type: av.AAC, Time: 50 * time.Second},
			},
			expected: false, // overlap=(10+10)s/45s=44%
		},
		{
			name:     "packets spanning multiple segments with majority overlap should skip",
			segments: [][2]float64{{10.0, 20.0}, {30.0, 40.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 12 * time.Second},
				{Type: av.AAC, Time: 38 * time.Second},
			},
			expected: true, // overlap=(8+8)s/26s=62%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adCutter := &AdCutter{
				segments: tt.segments,
			}

			result := adCutter.skipPackets(tt.packets)

			if result != tt.expected {
				t.Errorf("skipPackets() = %v, expected %v", result, tt.expected)
				t.Logf("Segments: %v", tt.segments)
				t.Logf("Packets: %v", formatPackets(tt.packets))
			}
		})
	}
}

func TestAdCutter_processGop(t *testing.T) {
	type wantPkt struct {
		typ  int
		time time.Duration
	}
	type step struct {
		video   app.Video
		packets []av.Packet
		want    []wantPkt // nil means expect no packets (dropped)
	}

	seg := func(start, end float64) app.Segment {
		return app.Segment{Segment: [2]float64{start, end}}
	}
	video := func(id int64, durationSecs int, segments ...app.Segment) app.Video {
		return app.Video{
			Id:       id,
			FileInfo: app.FileInfo{DurationV: durationSecs},
			SbInfo:   app.SponsorBlockInfo{Segments: segments},
		}
	}

	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "no segments - packets pass through unchanged",
			steps: []step{
				{
					video: video(1, 100),
					packets: []av.Packet{
						{Type: av.H264, Time: 5 * time.Second},
						{Type: av.AAC, Time: 8 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 5 * time.Second},
						{av.AAC, 8 * time.Second},
					},
				},
			},
		},
		{
			name: "gop in segment dropped to config packets only",
			steps: []step{
				{
					video: video(1, 100, seg(10, 20)),
					packets: []av.Packet{
						{Type: av.H264DecoderConfig},
						{Type: av.H264, Time: 12 * time.Second},
						{Type: av.AACDecoderConfig},
						{Type: av.AAC, Time: 15 * time.Second},
					},
					want: []wantPkt{
						{av.H264DecoderConfig, 0},
						{av.AACDecoderConfig, 0},
					},
				},
			},
		},
		{
			name: "time adjusted after dropped segment",
			steps: []step{
				{
					// dropped: calcDuration = 15s-12s = 3s → timeOffset = 3s
					video: video(1, 100, seg(10, 20)),
					packets: []av.Packet{
						{Type: av.H264, Time: 12 * time.Second},
						{Type: av.AAC, Time: 15 * time.Second},
					},
					want: nil,
				},
				{
					// kept: H264 25s-3s=22s, AAC 28s-3s=25s
					video: video(1, 100, seg(10, 20)),
					packets: []av.Packet{
						{Type: av.H264, Time: 25 * time.Second},
						{Type: av.AAC, Time: 28 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 22 * time.Second},
						{av.AAC, 25 * time.Second},
					},
				},
			},
		},
		{
			name: "multiple consecutive drops accumulate offset",
			steps: []step{
				{
					// drop 1: timeOffset = 15s-12s = 3s
					video: video(1, 100, seg(10, 30)),
					packets: []av.Packet{
						{Type: av.H264, Time: 12 * time.Second},
						{Type: av.AAC, Time: 15 * time.Second},
					},
					want: nil,
				},
				{
					// drop 2: timeOffset += 23s-20s = 3s → total 6s
					video: video(1, 100, seg(10, 30)),
					packets: []av.Packet{
						{Type: av.H264, Time: 20 * time.Second},
						{Type: av.AAC, Time: 23 * time.Second},
					},
					want: nil,
				},
				{
					// kept: H264 35s-6s=29s, AAC 38s-6s=32s
					video: video(1, 100, seg(10, 30)),
					packets: []av.Packet{
						{Type: av.H264, Time: 35 * time.Second},
						{Type: av.AAC, Time: 38 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 29 * time.Second},
						{av.AAC, 32 * time.Second},
					},
				},
			},
		},
		{
			name: "two segments with content before between and after",
			steps: []step{
				{
					// before first segment: kept, timeOffset=0, no adjustment
					video: video(1, 100, seg(10, 20), seg(30, 40)),
					packets: []av.Packet{
						{Type: av.H264, Time: 5 * time.Second},
						{Type: av.AAC, Time: 8 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 5 * time.Second},
						{av.AAC, 8 * time.Second},
					},
				},
				{
					// in first segment [10,20]: dropped, timeOffset = 15s-12s = 3s
					video: video(1, 100, seg(10, 20), seg(30, 40)),
					packets: []av.Packet{
						{Type: av.H264, Time: 12 * time.Second},
						{Type: av.AAC, Time: 15 * time.Second},
					},
					want: nil,
				},
				{
					// between segments: kept, H264 22s-3s=19s, AAC 25s-3s=22s
					video: video(1, 100, seg(10, 20), seg(30, 40)),
					packets: []av.Packet{
						{Type: av.H264, Time: 22 * time.Second},
						{Type: av.AAC, Time: 25 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 19 * time.Second},
						{av.AAC, 22 * time.Second},
					},
				},
				{
					// in second segment [30,40]: dropped, timeOffset += 35s-32s = 3s → 6s
					video: video(1, 100, seg(10, 20), seg(30, 40)),
					packets: []av.Packet{
						{Type: av.H264, Time: 32 * time.Second},
						{Type: av.AAC, Time: 35 * time.Second},
					},
					want: nil,
				},
				{
					// after both segments: kept, H264 45s-6s=39s, AAC 48s-6s=42s
					video: video(1, 100, seg(10, 20), seg(30, 40)),
					packets: []av.Packet{
						{Type: av.H264, Time: 45 * time.Second},
						{Type: av.AAC, Time: 48 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 39 * time.Second},
						{av.AAC, 42 * time.Second},
					},
				},
			},
		},
		{
			name: "new video resets timeOffset and segments",
			steps: []step{
				{
					// video 1: drop GOP → timeOffset = 3s
					video: video(1, 100, seg(10, 20)),
					packets: []av.Packet{
						{Type: av.H264, Time: 12 * time.Second},
						{Type: av.AAC, Time: 15 * time.Second},
					},
					want: nil,
				},
				{
					// video 2: no segments, timeOffset reset → packets unchanged
					video: video(2, 100),
					packets: []av.Packet{
						{Type: av.H264, Time: 12 * time.Second},
						{Type: av.AAC, Time: 15 * time.Second},
					},
					want: []wantPkt{
						{av.H264, 12 * time.Second},
						{av.AAC, 15 * time.Second},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := &AdCutter{}
			for i, s := range tt.steps {
				t.Run(fmt.Sprintf("step_%d", i), func(t *testing.T) {
					result := ac.processGop(gop{video: s.video, packets: s.packets})
					if len(result.packets) != len(s.want) {
						t.Fatalf("got %d packets, want %d\ngot:  %v\nwant: %v",
							len(result.packets), len(s.want),
							formatPackets(result.packets), s.want)
					}
					for j, w := range s.want {
						p := result.packets[j]
						if p.Type != w.typ || p.Time != w.time {
							t.Errorf("packet %d: got {Type:%d Time:%s}, want {Type:%d Time:%s}",
								j, p.Type, p.Time, w.typ, w.time)
						}
					}
				})
			}
		})
	}
}

// Helper function to format packets for logging
func formatPackets(packets []av.Packet) []string {
	result := make([]string, len(packets))
	for i, p := range packets {
		result[i] = fmt.Sprintf("Type:%d Time:%.3fs", p.Type, p.Time.Seconds())
	}
	return result
}
