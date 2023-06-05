package example

import (
	"bytes"
	"encoding/binary"
	"fmt"
	message2 "github.com/jzyong/TcpMonitor/service/gate/message"
	log2 "github.com/jzyong/golib/log"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"testing"
	"time"
)

// 启动客户端
func TestClient(t *testing.T) {

	go startClient("test1", "123", "test1")
	go startClient("test2", "123", "test2")
	select {
	case <-time.After(time.Minute * 10):
		return
	}
}

func startClient(account, password, imei string) {
	conn, err := net.Dial("tcp", "127.0.0.1:7010")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	//defer conn.Close()

	var seq int32 = 0

	//登录消息
	seq++
	loginRequest := &message2.UserLoginRequest{Account: account, Password: password, Imei: imei}
	sendMsg(conn, loginRequest, message2.MID_UserLoginReq, seq)

	//心跳消息
	go func(c net.Conn) {
		for seq < 10 {
			seq++
			heartRequest := &message2.HeartRequest{}
			sendMsg(c, heartRequest, message2.MID_HeartReq, seq)
			time.Sleep(time.Second * 2)
		}
		c.Close() //关闭连接
	}(conn)

	go switchReceiveMessage(conn)

}

// 分发接收消息
func switchReceiveMessage(conn net.Conn) {
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
		case int32(message2.MID_UserLoginRes):
			var loginResponse message2.UserLoginResponse
			proto.Unmarshal(data, &loginResponse)
			log2.Info("用户登录返回：%v ", loginResponse)
			break
		case int32(message2.MID_ReconnectRes):

			break
		case int32(message2.MID_HeartRes):
			heartResponse := &message2.HeartResponse{}
			proto.Unmarshal(data, heartResponse)
			log2.Info("心跳返回：%v", heartResponse)
			break
		}
	}

}
