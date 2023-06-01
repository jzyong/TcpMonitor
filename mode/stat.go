package mode

import (
	"fmt"
	"github.com/google/gopacket"
)

const ExpireDay int = 30          //数据保存天数
const DaySecond int32 = 24 * 3600 //一天秒数

// PacketMessage 包装配后的消息
type PacketMessage struct {
	NetFlow gopacket.Flow //网络层流向
	TcpFlow gopacket.Flow //TCP流向
	Message interface{}   //自定义消息
}

// SocketStat 用户socket连接统计，创建一个socket就创建一个对象，存mongodb
type SocketStat struct {
	Id             string       `_id`           //ID
	Connection     Connection   `connection`    //连接
	BeginTime      int64        `beginTime`     //连接开始时间（不一定是创建，可能截取不到三次握手） ms
	EndTime        int64        `endTime`       //连接结束时间 ms
	ActiveTime     int64        `activeTime`    //活跃时间，即最后收到消息包的时间 ms
	Rtt            uint32       `rtt`           //往返时间  通过服务器推送消息包然后获得客户端都ack包进行确认，只统计一次
	MinTTL         uint8        `minTTL`        //最小ttl
	MinWindowSize  uint16       `minWindowSize` //最小窗口大小
	SYN            bool         `syn`           //是否截取到syn标识
	FINClient      bool         `finClient`     //客户端请求结束
	FINServer      bool         `finServer`     //服务器请求结束
	RSTClient      bool         `rstClient`     //客户端异常结束
	RSTServer      bool         `rstServer`     //服务器异常结束
	PacketStat     *PacketStat  `packetStat`    //包统计
	MessageStat    *MessageStat `messageStat`   //消息统计
	messages       []any        //序列化后的自定义消息,单独存储
	rttCaptureTime int64        //Rtt捕获时间
}

func (s *SocketStat) GetMessages() *[]any {
	return &s.messages
}

func (s *SocketStat) SetMessages(messages *[]any) {
	s.messages = *messages
}

func (s *SocketStat) GetRttCaptureTime() int64 {
	return s.rttCaptureTime
}

func (s *SocketStat) SetRttCaptureTime(rttCaptureTime int64) {
	s.rttCaptureTime = rttCaptureTime
}

// DurationSecond 持续秒
func (s *SocketStat) DurationSecond() int32 {
	if s.ActiveTime > s.BeginTime {
		return int32((s.ActiveTime - s.BeginTime) / 1000)
	}
	return 0
}

// PacketStat 包统计
type PacketStat struct {
	UploadPackets             int64  `uploadPackets`   //上行包个数
	DownloadPackets           int64  `downloadPackets` //下行包个数
	UploadBytes               int64  `uploadBytes`     //上行包个数
	DownloadBytes             int64  `downloadBytes`   //下行包个数
	UploadRetransmissionCount uint32 //上行重传次数
	DownRetransmissionCount   uint32 //上行重传次数
	uploadExpectSeq           uint32 //上行期待序列号
	downExpectSeq             uint32 //上行期待序列号
}

// Add 累加
func (s *PacketStat) Add(stat *PacketStat) {
	s.UploadPackets += stat.UploadPackets
	s.DownloadPackets += stat.DownloadPackets
	s.UploadBytes += stat.UploadBytes
	s.DownloadBytes += stat.DownloadBytes
	s.UploadRetransmissionCount += stat.UploadRetransmissionCount
	s.DownRetransmissionCount += stat.DownRetransmissionCount
}

// RetransmissionRate 重传率
func (s *PacketStat) RetransmissionRate() float32 {
	totalPacket := s.UploadPackets + s.DownloadPackets
	if totalPacket == 0 {
		return 0
	}
	retransmissionCount := s.UploadRetransmissionCount + s.DownRetransmissionCount
	return float32(retransmissionCount) / float32(totalPacket)
}

func (s *PacketStat) UploadRetransmissionRate() float32 {
	if s.UploadPackets == 0 {
		return 0
	}
	return float32(s.UploadRetransmissionCount) / float32(s.UploadPackets)
}

func (s *PacketStat) DownloadRetransmissionRate() float32 {
	if s.DownloadPackets == 0 {
		return 0
	}
	return float32(s.DownRetransmissionCount) / float32(s.DownloadPackets)
}

func (s *PacketStat) GetUploadExpectSeq() uint32 {
	return s.uploadExpectSeq
}
func (s *PacketStat) SetUploadExpectSeq(seq uint32) {
	s.uploadExpectSeq = seq
}

func (s *PacketStat) GetDownExpectSeq() uint32 {
	return s.downExpectSeq
}
func (s *PacketStat) SetDownExpectSeq(seq uint32) {
	s.downExpectSeq = seq
}

// UploadRps 上行rps
func (s *PacketStat) UploadRps(second int32) float32 {
	if second == 0 {
		return 0
	}
	return float32(s.UploadPackets) / float32(second)
}

// DownloadRps 下行rps
func (s *PacketStat) DownloadRps(second int32) float32 {
	if second == 0 {
		return 0
	}
	return float32(s.DownloadPackets) / float32(second)
}

// UploadBytesAverage 上行平均大小
func (s *PacketStat) UploadBytesAverage() int64 {
	if s.UploadPackets == 0 {
		return 0
	}
	return s.UploadBytes / s.UploadPackets
}

// DownloadBytesAverage 下行平均大小
func (s *PacketStat) DownloadBytesAverage() int64 {
	if s.DownloadPackets == 0 {
		return 0
	}
	return s.DownloadBytes / s.DownloadPackets
}

// MessageStat 组装好的消息统计
type MessageStat struct {
	UploadCount   int64  `uploadCount`   //上行包个数
	DownloadCount int64  `downloadCount` //下行包个数
	UploadBytes   int64  `uploadBytes`   //上行包个数
	DownloadBytes int64  `downloadBytes` //下行包个数
	MaxBytes      uint32 `maxBytes`      //最大消息包
	AppStat       any    `appStat`       //app自定义统计
}

// UploadRps 上行rps
func (s *MessageStat) UploadRps(second int32) float32 {
	if second == 0 {
		return 0
	}
	return float32(s.UploadCount) / float32(second)
}

// DownloadRps 下行rps
func (s *MessageStat) DownloadRps(second int32) float32 {
	if second == 0 {
		return 0
	}
	return float32(s.DownloadCount) / float32(second)
}

// Rps 总rps
func (s *MessageStat) Rps(second int32) float32 {
	if second == 0 {
		return 0
	}
	return float32(s.DownloadCount+s.UploadCount) / float32(second)
}

// UploadBytesAverage 上行平均大小
func (s *MessageStat) UploadBytesAverage() int64 {
	if s.UploadCount == 0 {
		return 0
	}
	return s.UploadBytes / s.UploadCount
}

// DownloadBytesAverage 下行平均大小
func (s *MessageStat) DownloadBytesAverage() int64 {
	if s.DownloadCount == 0 {
		return 0
	}
	return s.DownloadBytes / s.DownloadCount
}

// Add 累加
func (s *MessageStat) Add(stat *MessageStat) {
	s.UploadCount += stat.UploadCount
	s.DownloadCount += stat.DownloadCount
	s.UploadBytes += stat.UploadBytes
	s.DownloadBytes += stat.DownloadBytes
	if stat.MaxBytes > s.MaxBytes {
		s.MaxBytes = stat.MaxBytes
	}
}

// Connection 连接
type Connection struct {
	Local  Address `local`  //本地地址
	Remote Address `remote` //远端地址
}

// 格式化输出
func (c *Connection) String() string {
	return fmt.Sprintf("%v:%v --> %v:%v", c.Local.Ip, c.Local.Port, c.Remote.Ip, c.Remote.Port)
}

// Address 地址
type Address struct {
	Ip   string `ip`   //ip地址
	Port uint16 `port` //端口
}
