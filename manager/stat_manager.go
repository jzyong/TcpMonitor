package manager

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"math"
	"sync"
	"time"
)

// StatManager  基础入口
type StatManager struct {
	util.DefaultModule
	packets     chan gopacket.Packet                 //原生的数据包
	Messages    chan *mode.PacketMessage             //序列化后的消息
	saveChannel chan *mode.SocketStat                //存储socket统计
	Connections map[mode.Connection]*mode.SocketStat //连接统计信息
	AppStatFun  AppStateFun                          //app自定义统计逻辑
	AppUpdate   func()                               //app自定义定时逻辑回调
}

// AppStateFun 自定义统计函数
type AppStateFun func(stat *mode.SocketStat, message *mode.PacketMessage, clientRequest bool)

var statManager *StatManager
var statSingletonOnce sync.Once

func GetStatManager() *StatManager {
	statSingletonOnce.Do(func() {
		statManager = &StatManager{
			packets:     make(chan gopacket.Packet, 1024),
			Messages:    make(chan *mode.PacketMessage, 1024),
			saveChannel: make(chan *mode.SocketStat, 1024),
			Connections: make(map[mode.Connection]*mode.SocketStat),
		}
	})
	return statManager
}

func (m *StatManager) Init() error {
	log.Info("StatManager init start......")

	go m.ProcessStats()
	go m.saveSocketStat()
	log.Info("StatManager started ......")

	return nil
}

func (m *StatManager) Run() {
}

func (m *StatManager) Stop() {
}

// ProcessStats Stats 进行统计
// 将Packet和PacketMessage传到一个routine中处理，避免并发问题
func (m *StatManager) ProcessStats() {
	ticker := time.Tick(time.Second * 5)
	for {
		select {
		case packet := <-m.packets:
			ipv4 := packet.NetworkLayer().(*layers.IPv4)
			srcIp := ipv4.SrcIP.String()
			dstIp := ipv4.DstIP.String()
			tcp := packet.TransportLayer().(*layers.TCP)
			srcPort := parsePort(tcp.SrcPort.String())
			dstPort := parsePort(tcp.DstPort.String())
			var connection mode.Connection
			clientRequest := true
			if m.serverPort(dstPort) {
				connection = mode.Connection{Local: mode.Address{Ip: srcIp, Port: srcPort}, Remote: mode.Address{Ip: dstIp, Port: dstPort}}
			} else {
				connection = mode.Connection{Remote: mode.Address{Ip: srcIp, Port: srcPort}, Local: mode.Address{Ip: dstIp, Port: dstPort}}
				clientRequest = false
			}
			socketStat := m.Connections[connection]
			if socketStat == nil {
				socketStat = &mode.SocketStat{Connection: connection, BeginTime: time.Now().UnixMilli(), PacketStat: &mode.PacketStat{}, MessageStat: &mode.MessageStat{}}
				socketStat.MinTTL = math.MaxUint8
				socketStat.MinWindowSize = math.MaxUint16
				m.Connections[connection] = socketStat
				logStr := fmt.Sprintf("连接创建：%v Syn=%v", connection, tcp.SYN)
				log.Info(logStr)
				GetConsoleManager().AddConsoleLog(mode.SocketCreate, connection.String(), logStr, "", nil)
			}
			if socketStat.MinTTL > ipv4.TTL {
				socketStat.MinTTL = ipv4.TTL
			}

			m.statPackets(socketStat, tcp, clientRequest)
		case message := <-m.Messages:
			// 注入自定义逻辑，但是又不能直接调用service逻辑
			if m.AppStatFun != nil {
				connection, clientRequest := m.getMessageConnection(message)
				socketStat := m.Connections[connection]
				if socketStat != nil {
					m.AppStatFun(socketStat, message, clientRequest)
				} else {
					log.Info("%v 获取连接失败", connection)
				}
			}
		case <-ticker:
			// 定时监测异常完成的消息包，并存数据库,清除完成的连接
			m.clearFinishConnection()
			if m.AppUpdate != nil {
				m.AppUpdate()
			}
		}
	}
}

// 存储 socketStat 单独用一个routine存储数据，防止阻塞其他计算，只能所有操作完成后进行
func (m *StatManager) saveSocketStat() {
	for {
		select {
		case socketStat := <-m.saveChannel:
			GetDataManager().InsertSocketStat(socketStat)
		}
	}
}

// 是否为服务器端口
func (m *StatManager) serverPort(port uint16) bool {
	for _, p := range config2.ApplicationConfigInstance.ServerPort {
		if p == port {
			return true
		}
	}
	return false
}

// 进行消息包统计
func (m *StatManager) statPackets(socketStat *mode.SocketStat, tcp *layers.TCP, clientRequest bool) {
	//收到结束标识后，后面的几个消息包直接忽略
	if socketStat.EndTime > 0 {
		return
	}
	if tcp.SYN {
		socketStat.SYN = tcp.SYN
	}
	if socketStat.MinWindowSize > tcp.Window {
		socketStat.MinWindowSize = tcp.Window
	}

	now := time.Now().UnixMilli()
	socketStat.ActiveTime = now
	packetStat := socketStat.PacketStat
	dataLength := len(tcp.Contents) + len(tcp.Payload)
	if clientRequest {
		packetStat.UploadPackets++
		packetStat.UploadBytes = packetStat.UploadBytes + int64(dataLength)
		//重传统计
		if tcp.Seq < packetStat.GetUploadExpectSeq() {
			packetStat.UploadRetransmissionCount++
		}
		packetStat.SetUploadExpectSeq(tcp.Seq + uint32(len(tcp.Payload)))
		//计算rtt
		if socketStat.Rtt < 1 && socketStat.GetRttCaptureTime() > 0 {
			socketStat.Rtt = uint32(now - socketStat.GetRttCaptureTime())
		}
	} else {
		packetStat.DownloadPackets++
		packetStat.DownloadBytes = packetStat.DownloadBytes + int64(dataLength)
		//重传统计
		if tcp.Seq < packetStat.GetDownExpectSeq() {
			packetStat.DownRetransmissionCount++
		}
		packetStat.SetDownExpectSeq(tcp.Seq + uint32(len(tcp.Payload)))

		//设置rtt捕获时间
		if socketStat.Rtt < 1 {
			socketStat.SetRttCaptureTime(now)
		}
	}

	//结束标志
	if tcp.FIN {
		socketStat.EndTime = now
		if clientRequest {
			socketStat.FINClient = true
		} else {
			socketStat.FINServer = true
		}
	}
	//异常结束
	if tcp.RST {
		socketStat.EndTime = now
		if clientRequest {
			socketStat.RSTClient = true
		} else {
			socketStat.RSTServer = true
		}
	}

	//log.Info("消息包统计：%v", util.ToString(socketStat.PacketStat))
}

// 获取消息包连接
// 连接，true 客户端请求
func (m *StatManager) getMessageConnection(message *mode.PacketMessage) (mode.Connection, bool) {
	portSrc, portDst := message.TcpFlow.Endpoints()
	ipSrc, ipDst := message.NetFlow.Endpoints()
	//log.Info("%v %v %v %v", ipSrc.String(), portSrc.String(), ipDst.String(), portDst.String())
	srcPort := parsePort(portSrc.String())
	dstPort := parsePort(portDst.String())
	if m.serverPort(dstPort) {
		return mode.Connection{Local: mode.Address{Ip: ipSrc.String(), Port: srcPort}, Remote: mode.Address{Ip: ipDst.String(), Port: dstPort}}, true
	} else {
		return mode.Connection{Remote: mode.Address{Ip: ipSrc.String(), Port: srcPort}, Local: mode.Address{Ip: ipDst.String(), Port: dstPort}}, false
	}
}

// 清理已经完成的socket连接
func (m *StatManager) clearFinishConnection() {
	now := time.Now().UnixMilli()
	for connection, socketStat := range m.Connections {
		//结束或网络超时
		if (socketStat.EndTime > 0 && now-socketStat.EndTime > 10000) || (now-socketStat.ActiveTime) > 600000 {
			logStr := fmt.Sprintf("连接关闭：Address=%v ClientFin=%v ClientRst=%v ServerFin=%v ServerRst=%v", socketStat.Connection, socketStat.FINClient, socketStat.RSTClient, socketStat.FINServer, socketStat.RSTServer)
			log.Info(logStr)
			GetConsoleManager().AddConsoleLog(mode.SocketClose, connection.String(), logStr, "", nil)
			// 存数据库,所有消息详细存数据库是否会太大，超过16M？统计查询一定不能包括具体的消息
			delete(m.Connections, connection)
			m.saveChannel <- socketStat
		}
	}
}
