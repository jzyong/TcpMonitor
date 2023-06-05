package example

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jzyong/TcpMonitor/service/gate/message"
	log2 "github.com/jzyong/golib/log"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

var userId int64 = 0

// 启动服务器
func TestSever(t *testing.T) {
	listener, err := net.Listen("tcp", ":7010")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err)
			continue
		}
		go handleConnection(conn)
	}
}

// 处理连接数据
func handleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		bufLength := make([]byte, 4)
		_, err := io.ReadFull(conn, bufLength)
		if err != nil {
			fmt.Println("Error reading:", err)
			return
		}
		var msgLength = uint32(binary.LittleEndian.Uint32(bufLength))

		msgData := make([]byte, msgLength)
		if _, err := io.ReadFull(conn, msgData); err != nil {
			fmt.Println("read msg data error ", err)
			return
		}

		dataBuff := bytes.NewReader(msgData)
		//读msgID
		var messageId int32
		if err := binary.Read(dataBuff, binary.LittleEndian, &messageId); err != nil {
			log2.Info("读取消息ID错误:%v", err)
			return
		}
		//读确认序列号
		var ack int32
		if err := binary.Read(dataBuff, binary.LittleEndian, &ack); err != nil {
			log2.Info("读取确认号错误:%v", err)
			return
		}
		//读 序列号
		var seq int32
		if err := binary.Read(dataBuff, binary.LittleEndian, &seq); err != nil {
			log2.Info("读取序列号错误:%v", err)
			return
		}
		//读取数据
		data := make([]byte, msgLength-12)
		if err := binary.Read(dataBuff, binary.LittleEndian, data); err != nil {
			log2.Info("读取proto数据错误:%v", err)
			return
		}
		switch messageId {
		case int32(message.MID_UserLoginReq):
			var loginRequest message.UserLoginRequest
			proto.Unmarshal(data, &loginRequest)

			atomic.AddInt64(&userId, 1)
			loginResponse := &message.UserLoginResponse{UserId: userId}
			log2.Info("用户登录：%+v ==>\r\n %v", loginRequest, loginResponse)
			sendMsg(conn, loginResponse, message.MID_UserLoginRes, seq)
			break
		case int32(message.MID_ReconnectReq):

			break
		case int32(message.MID_HeartReq):
			heartResponse := &message.HeartResponse{TimeStamp: time.Now().UnixMilli()}
			data, err = proto.Marshal(heartResponse)
			if err != nil {
				fmt.Println("Error marshalling:", err)
				return
			}
			log2.Info("心跳返回：%v", heartResponse)
			sendMsg(conn, heartResponse, message.MID_HeartRes, seq)
			break
		}
	}

}

// 发送消息
func sendMsg(conn net.Conn, message proto.Message, mid message.MID, seq int32) {
	data, err := proto.Marshal(message)
	if err != nil {
		log2.Error("Error marshalling:%v", err)
		return
	}

	//创建一个存放bytes字节的缓冲
	dataBuff := bytes.NewBuffer([]byte{})

	//写dataLen 不包含自身长度
	var length int32 = int32(len(data)) + 12
	if err := binary.Write(dataBuff, binary.LittleEndian, length); err != nil {
		log2.Error("%v", err)
		return
	}
	//写msgID
	if err := binary.Write(dataBuff, binary.LittleEndian, int32(mid)); err != nil {
		log2.Error("%v", err)
		return
	}
	//写ack
	if err := binary.Write(dataBuff, binary.LittleEndian, int32(0)); err != nil {
		log2.Error("%v", err)
		return
	}
	//写seq
	if err := binary.Write(dataBuff, binary.LittleEndian, seq); err != nil {
		log2.Error("%v", err)
		return
	}
	//写data数据
	if err := binary.Write(dataBuff, binary.LittleEndian, data); err != nil {
		log2.Error("%v", err)
		return
	}
	conn.Write(dataBuff.Bytes())
}
