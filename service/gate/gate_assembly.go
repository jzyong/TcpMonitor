package gate

import (
	"bytes"
	"encoding/binary"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/mode"
	message2 "github.com/jzyong/TcpMonitor/service/gate/message"
	log2 "github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"google.golang.org/protobuf/proto"
	"io"
	"time"
)

//装配的消息不是有序的，展示列表顺序不可信

// GateStreamFactory 装配 slots-gate 客户端格式消息
// 参考 gopacket/examples/httpassembly
// StreamPool有缓存，每次两倍增长，且不会回收
type GateStreamFactory struct {
}

// GateStream 处理收到的消息包
type GateStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
}

func (h *GateStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	stream := &GateStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
	}
	go stream.run() // Important... we must guarantee that data from the reader stream is read.

	// ReaderStream implements tcpassembly.Stream, so we can return a pointer to it.
	return &stream.r
}

// 装配后的数据并不一定是有序的
func (s *GateStream) run() {
	//reader := bufio.NewReader(&s.r)
	reader := &s.r
	for {
		msgLengthBytes := make([]byte, 4)
		_, err := io.ReadFull(reader, msgLengthBytes)
		if err == io.EOF {
			// We must read until we see an EOF... very important!
			//log2.Info("EOF")
			return
		} else if err != nil {
			log2.Error("读取长度流错误%v %v：%v", s.net, s.transport, err)
			break
		}
		var msgLength = binary.LittleEndian.Uint32(msgLengthBytes)
		//前面是标识位
		msgLength = msgLength & 0xFFFFF
		msgData := make([]byte, msgLength)
		if _, err := io.ReadFull(reader, msgData); err != nil && err != io.EOF {
			log2.Error("读取数据流错误%v %v：长度=%v %v", s.net, s.transport, msgLength, err)
			break
		}

		dataBuff := bytes.NewReader(msgData)
		//读msgID
		var messageId int32
		if err := binary.Read(dataBuff, binary.LittleEndian, &messageId); err != nil {
			log2.Info("读取消息ID错误:%v", err)
			continue
		}
		//读确认序列号
		var ack int32
		if err := binary.Read(dataBuff, binary.LittleEndian, &ack); err != nil {
			log2.Info("读取确认号错误:%v", err)
			continue
		}
		//读 序列号
		var seq int32
		if err := binary.Read(dataBuff, binary.LittleEndian, &seq); err != nil {
			log2.Info("读取序列号错误:%v", err)
			continue
		}
		//读取数据
		data := make([]byte, msgLength-12)
		if err := binary.Read(dataBuff, binary.LittleEndian, data); err != nil {
			log2.Info("读取proto数据错误:%v", err)
			continue
		}

		gateMessage := &GateMessage{MessageId: messageId, Seq: seq, Ack: ack, Length: msgLength, Bytes: data, Time: time.Now().UnixMilli()}

		if config2.ApplicationConfigInstance.UnmarshalPacket {
			message := MessageDecoder[messageId]
			if message != nil {
				proto.Unmarshal(data, message)
				log2.Debug("截取消息包 ID=%v ACK=%v SEQ=%v LENGTH=%v %v \r\n%v", messageId, ack, seq, msgLength, message2.MID(messageId), util.ToStringIndent(message))
			} else {
				log2.Debug("截取消息包 ID=%v ACK=%v SEQ=%v LENGTH=%v", messageId, ack, seq, msgLength)
			}
		}
		packetMessage := &mode.PacketMessage{
			Message: gateMessage,
			NetFlow: s.net,
			TcpFlow: s.transport,
		}
		manager.GetStatManager().Messages <- packetMessage
	}
}
