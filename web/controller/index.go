package controller

import (
	"context"
	"fmt"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/TcpMonitor/service/gate"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math"
	"runtime"
	"strconv"
	"time"
)

// IndexController 首页
type IndexController struct {
	BaseController
}

// Index 首页
// http://localhost:5041
func (m *IndexController) Index() {
	m.Data["data"] = getData()
	m.display("index/index.html")
}

// 获取首页统计数据
func getData() map[string]interface{} {
	data := make(map[string]interface{})

	//查询数据
	collection := manager.GetDataManager().GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//只统计一天的数据
	beginTime := util.CurrentTimeMillisecond() - 24*3600*1000
	query := bson.M{"beginTime": bson.M{"$gt": beginTime}}
	cur, err := collection.Find(ctx, query)
	socketStats := make([]*mode.SocketStat, 0, 1000)
	if err != nil {
		log.Error("%v", err)
		return data
	}
	srcIpMap := make(map[string]bool)
	stat := &IndexStat{CloseStat: &CloseStat{}, DurationStat: &DurationStat{}, MessageCountStat: &MessageCountStat{}}
	stat.PacketStat = &mode.PacketStat{}
	stat.MessageStat = &mode.MessageStat{}
	stat.MessageStat.AppStat = &gate.GateStat{}
	for cur.Next(ctx) {
		var socketStat = &mode.SocketStat{}
		err := cur.Decode(&socketStat)
		if err != nil {
			log.Fatal("%v", err)
		}
		socketStats = append(socketStats, socketStat)
		srcIpMap[socketStat.Connection.Local.Ip] = true
		stat.PacketStat.Add(socketStat.PacketStat)
		stat.MessageStat.Add(socketStat.MessageStat)
		stat.Add(socketStat)
	}
	//第一行
	//关键数据统计
	data["socketCount"] = len(socketStats)
	data["ipCount"] = len(srcIpMap)
	data["packetRetransmissionRate"] = fmt.Sprintf("%.2f", stat.PacketStat.RetransmissionRate()*100)
	data["messageRetransmissionRate"] = fmt.Sprintf("%.2f", stat.RetransmissionRate()*100)
	data["tcpConnectCount"] = len(manager.GetStatManager().Connections)
	if len(socketStats) < 1 {
		data["tcpConnectTimeAverage"] = 0
		data["packetRtt"] = 0
	} else {
		data["tcpConnectTimeAverage"] = stat.DurationSecond / int32(len(socketStats))
		data["packetRtt"] = stat.Rtt / uint32(len(socketStats))
	}
	data["tcpConnectTime"] = stat.DurationSecond

	//第二行
	//应用统计
	data["appTimeOutCount"] = stat.TimeOutCount
	data["appSeqExecuteTime"] = stat.SeqExecuteTime / 1000
	data["appSeqExecuteTimeAverage"] = stat.executeAverageTime()
	data["appSeqCount"] = stat.SeqCount
	data["appLoginCount"] = stat.LoginCount
	data["appReconnectCount"] = stat.ReconnectCount
	data["tcpConnectTimeMax"] = stat.MaxConnectTime
	//消息统计
	data["messageUploadCount"] = stat.MessageStat.UploadCount
	data["messageDownloadCount"] = stat.MessageStat.DownloadCount
	data["messageUploadBytes"] = stat.MessageStat.UploadBytes
	data["messageDownloadBytes"] = stat.MessageStat.DownloadBytes
	data["messageUploadBytesAverage"] = stat.MessageStat.UploadBytesAverage()
	data["messageDownloadBytesAverage"] = stat.MessageStat.DownloadBytesAverage()
	data["messageUploadRps"] = fmt.Sprintf("%.2f/%.2f", stat.MessageStat.UploadRps(stat.DurationSecond), stat.MessageStat.UploadRps(mode.DaySecond))
	data["messageDownloadRps"] = fmt.Sprintf("%.2f/%.2f", stat.MessageStat.DownloadRps(stat.DurationSecond), stat.MessageStat.DownloadRps(mode.DaySecond))
	//包统计
	data["packetUploadCount"] = stat.PacketStat.UploadPackets
	data["packetDownloadCount"] = stat.PacketStat.DownloadPackets
	data["packetUploadBytes"] = stat.PacketStat.UploadBytes
	data["packetDownloadBytes"] = stat.PacketStat.DownloadBytes
	data["packetUploadBytesAverage"] = stat.PacketStat.UploadBytesAverage()
	data["packetDownloadBytesAverage"] = stat.PacketStat.DownloadBytesAverage()
	data["packetUploadRps"] = fmt.Sprintf("%.2f/%.2f", stat.PacketStat.UploadRps(stat.DurationSecond), stat.PacketStat.UploadRps(mode.DaySecond))
	data["packetDownloadRps"] = fmt.Sprintf("%.2f/%.2f", stat.PacketStat.DownloadRps(stat.DurationSecond), stat.PacketStat.DownloadRps(mode.DaySecond))
	data["packetRetransmission"] = fmt.Sprintf("%.2f%%/%.2f%%", stat.PacketStat.UploadRetransmissionRate()*100, stat.PacketStat.DownloadRetransmissionRate()*100)
	//系统统计
	cpuPercent, _ := cpu.Percent(0, true)
	var cpuAll float64
	for _, v := range cpuPercent {
		cpuAll += v
	}
	loads, _ := load.Avg()
	data["load"] = loads.String()
	data["cpu"] = math.Round(cpuAll / float64(len(cpuPercent)))
	vir, _ := mem.VirtualMemory()
	data["virtual_mem"] = math.Round(vir.UsedPercent)
	io1, _ := net.IOCounters(false)
	conn, _ := net.ProtoCounters(nil)
	time.Sleep(time.Millisecond * 100) //需要等待时间拉取统计数据，降低了反应时间
	io2, _ := net.IOCounters(false)
	if len(io2) > 0 && len(io1) > 0 {
		data["io_send"] = (io2[0].BytesSent - io1[0].BytesSent) * 10
		data["io_recv"] = (io2[0].BytesRecv - io1[0].BytesRecv) * 10
		packetSend := (io2[0].PacketsSent - io1[0].PacketsSent) * 10
		packetRecv := (io2[0].PacketsRecv - io1[0].PacketsRecv) * 10
		data["io_count"] = fmt.Sprintf("%v/%v", packetRecv, packetSend)

	}
	data["routineCount"] = runtime.NumGoroutine()

	for _, v := range conn {
		data[v.Protocol] = v.Stats["CurrEstab"]
		log.Debug("连接建立：%v=%v", v.Protocol, v.Stats["CurrEstab"])
	}

	//第三行及第四行
	//rps，连接，带宽
	var fg int
	if len(gate.IndexLines) > 0 {
		fg = len(gate.IndexLines) / 12
		for i := 0; i <= 11; i++ {
			data["indexLine"+strconv.Itoa(i+1)] = gate.IndexLines[i*fg]
		}
	}

	//第五行
	//关闭类型
	data["closeType"] = stat.CloseStat
	//连接类型
	data["loginCount"] = stat.LoginCount
	data["reconnectCount"] = stat.ReconnectCount
	//时长
	data["durationStat"] = stat.DurationStat
	//消息数
	data["messageCountStat"] = stat.MessageCountStat

	return data
}

// IndexStat 首页统计
type IndexStat struct {
	mode.SocketStat
	DurationSecond   int32             //持续时间
	MaxConnectTime   int32             //最大连接时间
	TimeOutCount     int32             //超时消息数
	SeqExecuteTime   int32             //带序列号消息总执行时间
	SeqCount         int32             //带序列号消息个数
	LoginCount       int32             //登录数
	ReconnectCount   int32             //重连数
	CloseStat        *CloseStat        //关闭类型技术
	DurationStat     *DurationStat     //时长统计
	MessageCountStat *MessageCountStat //消息个数统计
}

// DurationStat 时长统计 分钟
type DurationStat struct {
	Zero  uint
	One   uint
	Ten   uint
	Sixty uint
}

// MessageCountStat 消息个数统计
type MessageCountStat struct {
	Zero        uint
	One         uint
	Hundred     uint
	Thousand    uint
	TenThousand uint
}

// CloseStat 关闭统计
type CloseStat struct {
	ClientFin uint
	ClientRst uint
	ServerFin uint
	ServerRst uint
	TimeOut   uint
}

// Add 累加
func (i *IndexStat) Add(stat *mode.SocketStat) {
	second := stat.DurationSecond()
	i.DurationSecond += second
	if second > i.MaxConnectTime {
		i.MaxConnectTime = second
	}

	i.Rtt += stat.Rtt
	//自定义统计数据
	appStat := stat.MessageStat.AppStat
	if appStat != nil {
		st := appStat.(primitive.D)
		m := st.Map()
		i.TimeOutCount += m["timeOutCount"].(int32)
		i.SeqExecuteTime += m["seqExecuteTime"].(int32)
		i.SeqCount += m["seqCount"].(int32)
		if m["reconnect"] != nil && m["reconnect"].(bool) {
			i.ReconnectCount++
		} else {
			i.LoginCount++
		}
	}
	//关闭类型计数
	if stat.FINClient {
		i.CloseStat.ClientFin++
	} else if stat.FINServer {
		i.CloseStat.ServerFin++
	} else if stat.RSTClient {
		i.CloseStat.ClientRst++
	} else if stat.RSTServer {
		i.CloseStat.ServerRst++
	} else {
		i.CloseStat.TimeOut++
	}
	//时长计数
	minute := second / 60
	if minute < 1 {
		i.DurationStat.Zero++
	} else if minute < 10 {
		i.DurationStat.One++
	} else if minute < 60 {
		i.DurationStat.Ten++
	} else {
		i.DurationStat.Sixty++
	}
	//消息计数 请求消息
	if stat.MessageStat.UploadCount < 1 {
		i.MessageCountStat.Zero++
	} else if stat.MessageStat.UploadCount < 100 {
		i.MessageCountStat.One++
	} else if stat.MessageStat.UploadCount < 1000 {
		i.MessageCountStat.Hundred++
	} else if stat.MessageStat.UploadCount < 10000 {
		i.MessageCountStat.Thousand++
	} else {
		i.MessageCountStat.TenThousand++
	}
}

// 平均执行时间
func (i *IndexStat) executeAverageTime() int32 {
	if i.SeqCount == 0 {
		return 0
	}
	return i.SeqExecuteTime / i.SeqCount
}

// RetransmissionRate 重传率
func (i *IndexStat) RetransmissionRate() float32 {
	if i.SeqCount == 0 {
		return 0
	}
	return float32(i.TimeOutCount) / float32(i.SeqCount)
}
