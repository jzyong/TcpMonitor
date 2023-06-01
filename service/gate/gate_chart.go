package gate

import (
	"fmt"
	"github.com/jzyong/TcpMonitor/manager"
	"time"
)

//首页图表展示统计，每1分钟统计一条数据，不保存，进程关闭丢失

var (
	indexLineStat *IndexLineStat //当前统计
	IndexLines    []*IndexLine
)

// IndexLineStat 首页统计数据
type IndexLineStat struct {
	Time          time.Time
	UploadCount   uint32
	DownCount     uint32
	UploadBytes   uint32
	DownloadBytes uint32
}

// IndexLine 首页线图展示结构体
type IndexLine struct {
	Time          string
	UploadRps     string
	DownloadRps   string
	UploadBytes   float32
	DownloadBytes float32
	ConnectCount  uint32
}

// 启动统计
func starIndexLineStat() {
	go func() {
		IndexLines = make([]*IndexLine, 0, 1500)
		indexLineStat = &IndexLineStat{Time: time.Now()}
		var duration float32 = 1
		for {
			length := len(IndexLines)
			if length < 12 {
				time.Sleep(time.Second)
			} else if length < 100 {
				time.Sleep(time.Second * 3)
				duration = 3
			} else {
				time.Sleep(time.Minute)
				duration = 60
			}
			if len(IndexLines) >= 1440 {
				IndexLines = IndexLines[1:]
			}
			IndexLines = append(IndexLines, indexLineStatToShow(indexLineStat, duration))
			indexLineStat = &IndexLineStat{Time: time.Now()}
		}

	}()
}

// indexLineStatToShow 统计数据转换为show ，默认60s统计一次
func indexLineStatToShow(stat *IndexLineStat, duration float32) *IndexLine {
	indexLine := &IndexLine{
		Time:          stat.Time.Format("15:04:05"),
		UploadRps:     fmt.Sprintf("%.2f", float32(stat.UploadCount)/duration),
		DownloadRps:   fmt.Sprintf("%.2f", float32(stat.DownCount)/duration),
		ConnectCount:  uint32(len(manager.GetStatManager().Connections)),
		UploadBytes:   float32(stat.UploadBytes) / duration,
		DownloadBytes: float32(stat.DownloadBytes) / duration,
	}
	//log.Info("IndexLine=%v", indexLine)
	return indexLine
}
