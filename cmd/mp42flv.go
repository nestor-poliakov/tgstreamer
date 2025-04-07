package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yapingcat/gomedia/go-flv"
	"github.com/yapingcat/gomedia/go-mp4"
)

func main() {
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
