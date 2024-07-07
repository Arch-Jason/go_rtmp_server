package main

import (
	"log"
	"os"
	"time"

	"github.com/grafov/m3u8"
	"github.com/nareix/joy4/format"
	"github.com/nareix/joy4/format/rtmp"
	"github.com/nareix/joy4/format/ts"
)

func main() {
	format.RegisterAll()

	server := &rtmp.Server{}

	server.HandlePublish = func(conn *rtmp.Conn) {
		log.Println("Stream is being published.")

		p, err := m3u8.NewMediaPlaylist(3, 3)
		if err != nil {
			log.Fatal("Failed to create a new media playlist:", err)
		}

		streams, err := conn.Streams()
		if err != nil {
			log.Fatal("Failed to get streams:", err)
		}

		var tsMuxer *ts.Muxer
		var tsFile *os.File
		var packetCount int
		const maxPacketsPerFile = 200
		var tsFileName string
		var tsFileNames []string

		for {
			pkt, err := conn.ReadPacket()
			if err != nil {
				log.Println("Failed to read packet:", err)
				break
			}

			if tsMuxer == nil || packetCount >= maxPacketsPerFile {
				if tsMuxer != nil {
					tsMuxer.WriteTrailer()
					tsFile.Close()
					log.Println("File closed:", tsFileName)

					p.AppendSegment(&m3u8.MediaSegment{
						SeqId: uint64(pkt.Idx),
						URI:   tsFileName,
						// Duration: 10,
					})

					// This is where the .m3u8 file is updated
					m3u8File, err := os.Create("./stream.m3u8")
					if err != nil {
						log.Fatal(err)
					}
					if _, err = p.Encode().WriteTo(m3u8File); err != nil {
						log.Fatal("Failed to write m3u8 file:", err)
					}
					m3u8File.Close()
					p.Remove() // 从播放列表中移除这个文件段的信息

					// remove old segment files
					if len(tsFileNames) > 3 {
						oldestFile := tsFileNames[0]
						tsFileNames = tsFileNames[1:]  // 先更新列表，移除最旧的文件名
						if err := os.Remove(oldestFile); err != nil {  // 然后尝试删除文件
							log.Println("Failed to delete old segment file:", oldestFile, err)
						} else {
							log.Println("Successfully deleted old segment file:", oldestFile)
						}
					}
					
				}

				tsFileName = "segment-" + time.Now().Format("20060102150405") + ".ts"
				tsFileNames = append(tsFileNames, tsFileName)
				tsFile, err = os.Create(tsFileName)
				if err != nil {
					log.Fatal("Failed to create segment file:", err)
				}
				tsMuxer = ts.NewMuxer(tsFile)
				if err = tsMuxer.WriteHeader(streams); err != nil {
					log.Fatal("Failed to write header:", err)
				}
				packetCount = 0
			}

			err = tsMuxer.WritePacket(pkt)
			if err != nil {
				log.Fatal("Failed to write packet:", err)
			}
			packetCount++
		}
	}

	log.Println("Starting RTMP server on :1935")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("RTMP server failed: %v", err)
	}
}
