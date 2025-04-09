package streamer

import (
	"fmt"
	"time"

	"github.com/nestor-poliakov/joy5/av"
	"github.com/yapingcat/gomedia/go-mp4"
)

// ConvertMP4PacketToFLV converts an MP4 packet to an FLV packet
// It handles both audio (AAC) and video (H264) packets
func ConvertMP4PacketToFLV(mp4Packet mp4.AVPacket) (av.Packet, error) {
	flvPacket := av.Packet{
		Data:  mp4Packet.Data,
		Time:  time.Duration(mp4Packet.Pts), // Using PTS for the presentation timestamp
		CTime: time.Duration(mp4Packet.Dts), // Using DTS for the composition timestamp
	}

	switch mp4Packet.Cid {
	case mp4.MP4_CODEC_AAC: // Assuming MP4_CODEC_AAC is defined in your code
		flvPacket.Type = 8 // Audio packet in FLV

		// Check if this is an AAC sequence header (typically at the beginning of the stream)
		// This would depend on your specific implementation, but typically
		// the first AAC packet contains the AudioSpecificConfig
		if len(mp4Packet.Data) > 0 && isAACSequenceHeader(mp4Packet.Data) {
			flvPacket.ASeqHdr = mp4Packet.Data
		}

	case mp4.MP4_CODEC_H264: // Assuming MP4_CODEC_H264 is defined in your code
		flvPacket.Type = 9 // Video packet in FLV

		// Check if this frame is a keyframe
		// This depends on your H264 implementation, but typically you check the NAL unit type
		flvPacket.IsKeyFrame = isKeyFrame(mp4Packet.Data)

		// Check if this is an SPS/PPS (sequence/picture parameter sets - typically at the beginning)
		if isH264SequenceHeader(mp4Packet.Data) {
			flvPacket.VSeqHdr = mp4Packet.Data
		}

	default:
		return flvPacket, fmt.Errorf("unsupported codec type: %v", mp4Packet.Cid)
	}

	return flvPacket, nil
}

// isAACSequenceHeader determines if the data contains an AAC sequence header
// The implementation depends on your AAC format details
func isAACSequenceHeader(data []byte) bool {
	// Typically, AAC sequence header has specific markers
	// This is a simplified check - adjust based on your format
	if len(data) < 2 {
		return false
	}

	// Check for AudioSpecificConfig marker
	// This is just an example - you'll need to adapt to your actual format
	return (data[0] & 0xF0) == 0xA0
}

// isKeyFrame determines if the H264 data represents a keyframe
// The implementation depends on your H264 format details
func isKeyFrame(data []byte) bool {
	// In H.264, NAL unit type 5 is an IDR picture (keyframe)
	// This is a simplified check - adjust based on your format
	if len(data) < 5 {
		return false
	}

	// Check for NAL unit type
	// This assumes AVCC format with NAL size prefixes
	// You may need to adjust based on your actual format
	nalType := data[4] & 0x1F
	return nalType == 5 // IDR frame is type 5
}

// isH264SequenceHeader determines if the data contains H264 sequence parameters
// The implementation depends on your H264 format details
func isH264SequenceHeader(data []byte) bool {
	// In H.264, NAL unit types 7 (SPS) and 8 (PPS) typically appear in sequence headers
	// This is a simplified check - adjust based on your format
	if len(data) < 5 {
		return false
	}

	// Check for SPS or PPS NAL units
	// This assumes AVCC format with NAL size prefixes
	// You may need to adjust based on your actual format
	nalType := data[4] & 0x1F
	return nalType == 7 || nalType == 8 // SPS is type 7, PPS is type 8
}
