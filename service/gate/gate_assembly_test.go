package gate

import (
	"encoding/json"
	"github.com/golang/snappy"
	"github.com/jzyong/golib/log"
	"testing"
)

// 测试snappy json 压缩
func TestSnappy(t *testing.T) {
	messages := make([]GateMessage, 0, 5)
	for i := 0; i < 5; i++ {
		message := GateMessage{MessageId: int32(i), Seq: int32(i), Ack: int32(i), Length: uint32(i)}
		message.Bytes = []byte{18, 25, 44, 54, 21, 9, 40}
		messages = append(messages, message)
	}
	bytes, err := json.Marshal(messages)
	if err != nil {
		log.Warn("转byte错误:%v", err)
	}
	encodeBuf := snappy.Encode(nil, bytes)
	log.Info("压缩前：%v,压缩后：%v", len(bytes), len(encodeBuf))
	decodeBuf, err := snappy.Decode(nil, encodeBuf)
	if err != nil {
		log.Warn("解byte错误:%v", err)
	}

	messages2 := make([]GateMessage, 0, 5)
	err = json.Unmarshal(decodeBuf, &messages2)
	if err != nil {
		log.Warn("解json错误:%v", err)
	}
	log.Info("json数组：%+v", messages2)
}
