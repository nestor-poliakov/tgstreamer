package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-flv"
	"github.com/yapingcat/gomedia/go-mp4"
)

func main() {
	convert2()
}

func convert1() {
	t := time.Now()
	defer func() {
		fmt.Println(time.Since(t))
	}()
	mp4filename := os.Args[1]

	f, err := os.Open(mp4filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	demuxer := mp4.CreateMp4Demuxer(f)

	infos, err := demuxer.ReadHead()
	if err != nil && err != io.EOF {
		fmt.Println("read head error:", err)
		return
	} else {
		b, _ := json.MarshalIndent(infos, "", "  ")
		fmt.Printf("track info: %+v\n", string(b))
	}
	newFileName := strings.ReplaceAll(mp4filename, ".mp4", ".flv")
	flvFile, _ := os.OpenFile(newFileName, os.O_CREATE|os.O_RDWR, 0644)
	defer flvFile.Close()
	fw := flv.CreateFlvWriter(flvFile)
	fw.WriteFlvHeader()
	flv.NewFlvMuxer(vid flv.FLV_VIDEO_CODEC_ID, aid flv.FLV_SOUND_FORMAT)
	for i := 0; ; i++ {
		pkg, err := demuxer.ReadPacket()
		if err != nil {
			fmt.Println(err)
			break
		}
		if i < 10 {
			fmt.Printf("track:%d,cid:%+v,pts:%d dts:%d\n", pkg.TrackId, pkg.Cid, pkg.Pts, pkg.Dts)
		}
		switch pkg.Cid {
		case mp4.MP4_CODEC_H264:
			err = fw.WriteH264(pkg.Data, uint32(pkg.Pts), uint32(pkg.Dts))
		case mp4.MP4_CODEC_AAC:
			err = fw.WriteAAC(pkg.Data, uint32(pkg.Pts), uint32(pkg.Dts))
		case mp4.MP4_CODEC_H265:
			err = fw.WriteH265(pkg.Data, uint32(pkg.Pts), uint32(pkg.Dts))
		default:
			err = fmt.Errorf("unknoun packet type %d", pkg.Cid)
		}
		if err != nil {
			fmt.Println("error processing packet: %s", err)
		}
	}
}

func convert2() {
	mp4filename := os.Args[1]
	mp4file, err := os.OpenFile(mp4filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer mp4file.Close()

	muxer, err := mp4.CreateMp4Muxer(mp4file)
	if err != nil {
		fmt.Println(err)
		return
	}
	hasVideo := false
	hasAudio := false
	var vtid uint32 = 0
	var atid uint32 = 0

	flvfilereader, _ := os.Open(strings.ReplaceAll(mp4filename, ".mp4", ".flv"))
	defer flvfilereader.Close()
	fr := flv.CreateFlvReader()

	fr.OnFrame = func(ci codec.CodecID, b []byte, pts, dts uint32) {
		if ci == codec.CODECID_AUDIO_AAC {
			if !hasAudio {
				atid = muxer.AddAudioTrack(mp4.MP4_CODEC_AAC)
				hasAudio = true
			}
			err := muxer.Write(atid, b, uint64(pts), uint64(dts))
			if err != nil {
				fmt.Println(err)
			}
		} else if ci == codec.CODECID_VIDEO_H264 {
			if !hasVideo {
				vtid = muxer.AddVideoTrack(mp4.MP4_CODEC_H264)
				hasVideo = true
			}
			err := muxer.Write(vtid, b, uint64(pts), uint64(dts))
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	cache := make([]byte, 4096)
	for {
		n, err := flvfilereader.Read(cache)
		if err != nil {
			fmt.Println(err)
			break
		}
		fr.Input(cache[0:n])
	}
	muxer.WriteTrailer()
}
