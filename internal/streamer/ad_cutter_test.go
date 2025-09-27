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
			name:     "packets overlapping start of segment should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 8 * time.Second},
				{Type: av.AAC, Time: 12 * time.Second},
			},
			expected: true,
		},
		{
			name:     "packets overlapping end of segment should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 15 * time.Second},
				{Type: av.AAC, Time: 25 * time.Second},
			},
			expected: true,
		},
		{
			name:     "packets spanning entire segment should skip",
			segments: [][2]float64{{10.0, 20.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 5 * time.Second},
				{Type: av.AAC, Time: 25 * time.Second},
			},
			expected: true,
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
			name:     "packets spanning multiple segments should skip",
			segments: [][2]float64{{10.0, 20.0}, {30.0, 40.0}},
			packets: []av.Packet{
				{Type: av.H264, Time: 15 * time.Second}, // overlaps first segment
				{Type: av.AAC, Time: 35 * time.Second},  // overlaps second segment
			},
			expected: true,
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

// Helper function to format packets for logging
func formatPackets(packets []av.Packet) []string {
	result := make([]string, len(packets))
	for i, p := range packets {
		result[i] = fmt.Sprintf("Type:%d Time:%.3fs", p.Type, p.Time.Seconds())
	}
	return result
}
