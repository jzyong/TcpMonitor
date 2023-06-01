package gate

import (
	"encoding/json"
	"fmt"
	"github.com/golang/snappy"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/mode"
	message2 "github.com/jzyong/TcpMonitor/service/gate/message"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"google.golang.org/protobuf/proto"
	"math"
	"time"
)

// GateMessageStats 缓存所有消息统计
var GateMessageStats = make(map[int32]*GateMessageStat, 1000)

// GateMessages 所有截取到的网关消息，存数据库
type GateMessages struct {
	Id         string         `_id`
	CreateTime int64          `createTime` //创建时间
	Message    []*GateMessage `message`    //所有消息
	Bytes      []byte         //所有byte数据
}

// GateMessage 网关消息，传递
type GateMessage struct {
	MessageId int32  `messageId` //消息ID
	Seq       int32  `seq`       //序列号
	Ack       int32  `ack`       //确认号
	Length    uint32 `length`    //长度
	Time      int64  `time`      //消息时间
	Bytes     []byte //消息内容
}

// GateMessageStat 网关消息统计,不存储数据库，统计每个socket消息汇总和者网关的消息汇总。
type GateMessageStat struct {
	MessageId           int32   `json:"messageId"`           //消息ID
	MessageName         string  `json:"messageName"`         //消息名称
	TotalTime           int     `json:"totalTime"`           //总共执行时间(ms)
	DelayAverage        int     `json:"delayAverage"`        //平均延迟(ms)
	DelayMin            int     `json:"delayMin"`            //最小延迟(ms)
	DelayMax            int     `json:"delayMax"`            //最大延迟(ms)
	Count               int     `json:"count"`               //请求|推送次数
	FailCount           int     `json:"failCount"`           //失败次数
	FailRate            float32 `json:"failRate"`            //失败率
	SuccessCount        int     `json:"successCount"`        //成功次数
	RequestRepeatCount  int     `json:"requestRepeatCount"`  //重复请求数
	ResponseRepeatCount int     `json:"responseRepeatCount"` //重复返回数
	SizeTotal           int     `json:"sizeTotal"`           //消息总大小
	SizeAverage         int     `json:"sizeAverage"`         //消息平均值
	SizeMin             int     `json:"sizeMin"`             //消息最小值
	SizeMax             int     `json:"sizeMax"`             //消息最大值
	StartTime           int64   `json:"startTime"`           //计时开始时间
	EndTime             int64   `json:"endTime"`             //计时结束时间
	Rps                 float32 `json:"rps"`                 //请求|推送速率
}

// GateStat 网关自定义统计,存数据库
type GateStat struct {
	TimeOutCount   int32           `timeOutCount`   //超时消息数
	SeqExecuteTime int32           `seqExecuteTime` //带序列号消息总执行时间
	SeqCount       int32           `seqCount`       //带序列号消息个数
	Reconnect      bool            `reconnect`      //是否为重连
	PlayerId       string          `playerId`       //玩家ID
	seqMap         map[int32]int64 // 消息序列号 key：序列号 value：时间，不存数据库
}

// 统计自定义数据
func processStat(stat *mode.SocketStat, message *mode.PacketMessage, clientRequest bool) {
	gateMessage := message.Message.(*GateMessage)
	gateMessages := *stat.GetMessages()
	if gateMessages == nil {
		gateMessages = make([]any, 0, 300)
	}
	gateMessages = append(gateMessages, *gateMessage)
	stat.SetMessages(&gateMessages)

	messageStat := stat.MessageStat
	if gateMessage.Length > messageStat.MaxBytes {
		messageStat.MaxBytes = gateMessage.Length
	}

	appStat := messageStat.AppStat
	if appStat == nil {
		appStat = &GateStat{seqMap: make(map[int32]int64, 300)}
		messageStat.AppStat = appStat
	}
	gateStat := appStat.(*GateStat)

	//添加日志统计 ,查询时再构建
	manager.GetConsoleManager().AddConsoleLog(mode.Message, stat.Connection.String(), "", gateStat.PlayerId, gateMessage)

	gameLogic(gateMessage, gateStat)

	if clientRequest {
		messageStat.UploadCount++
		messageStat.UploadBytes += int64(gateMessage.Length)
		indexLineStat.UploadCount++
		indexLineStat.UploadBytes += gateMessage.Length
		if gateMessage.Seq > 0 {
			gateStat.SeqCount++
			//超时重传消息已第一次请求为准
			if _, ok := gateStat.seqMap[gateMessage.Seq]; ok {
				gateStat.TimeOutCount++
			} else {
				gateStat.seqMap[gateMessage.Seq] = time.Now().UnixMilli()
			}
		}
		if gateMessage.MessageId == int32(message2.MID_ReconnectReq) {
			gateStat.Reconnect = true
		}

	} else {
		messageStat.DownloadCount++
		messageStat.DownloadBytes += int64(gateMessage.Length)
		indexLineStat.DownCount++
		indexLineStat.DownloadBytes += gateMessage.Length
		if gateMessage.Seq > 0 {
			if t, ok := gateStat.seqMap[gateMessage.Seq]; ok {
				executeTime := time.Now().UnixMilli() - t
				gateStat.SeqExecuteTime += int32(executeTime)
				delete(gateStat.seqMap, gateMessage.Seq)
			}
		}
	}
	//log.Info("消息统计：%v", util.ToString(stat.MessageStat))
}

// 游戏逻辑
func gameLogic(gateMessage *GateMessage, gateStat *GateStat) {
	//重连，登录消息加密了
	mid := message2.MID(gateMessage.MessageId)
	switch mid {
	case message2.MID_UserLoginRes:
		protoMsg := MessageDecoder[gateMessage.MessageId]
		proto.Unmarshal(gateMessage.Bytes, protoMsg)
		msg2 := protoMsg.(*message2.UserLoginResponse)
		gateStat.PlayerId = fmt.Sprintf("%v", msg2.UserId)
		log.Debug("从登录消息获取玩家id=%v", gateStat.PlayerId)
	case message2.MID_ReconnectRes:
		protoMsg := MessageDecoder[gateMessage.MessageId]
		proto.Unmarshal(gateMessage.Bytes, protoMsg)
		msg2 := protoMsg.(*message2.ReconnectResponse)
		gateStat.PlayerId = fmt.Sprintf("%v", msg2.GetUserId())
		log.Debug("从重连消息获取玩家id=%v", gateStat.PlayerId)
	}

}

// 构建存储数据
func newMessages(id string, stat *mode.SocketStat) any {
	messages := make([]*GateMessage, 0, len(*stat.GetMessages()))
	for _, m := range *stat.GetMessages() {
		message := m.(GateMessage)
		messages = append(messages, &message)
	}
	bytes, err := json.Marshal(messages)
	if err != nil {
		log.Warn("转byte错误:%v", err)
	}
	//使用 snappy压缩后平均大小缩减了50%，从450k左右降到220k左右，但是存储空间却任然没有减少， 13k条数据仍然占用1.98G的空间，估计是mongodb使用量snappy进行压缩
	encodeBuf := snappy.Encode(nil, bytes)
	gateMessages := &GateMessages{Bytes: encodeBuf, Id: id, CreateTime: util.CurrentTimeMillisecond()}

	//计算所有消息统计
	calculateAllMessageStat(stat, messages)

	return gateMessages
}

// CalculateMessageStat 计算消息统计
func CalculateMessageStat(gateMessages []*GateMessage) []*GateMessageStat {
	messageStatMap := make(map[int32]*GateMessageStat, len(gateMessages)/4)
	//序列号消息
	type SeqMessage struct {
		RequestCount  int   //请求次数
		ResponseCount int   //返回次数
		RequestTime   int64 //请求时间
		MessageId     int32 //消息id，请求消息
		TotalTime     int   //总共执行时间(ms)
		DelayMin      int   //最小延迟(ms)
		DelayMax      int   //最大延迟(ms)

	}
	seqStatMap := make(map[int32]*SeqMessage, len(gateMessages)/3)

	for _, gateMessage := range gateMessages {
		//统计 基础信息
		messageStat := messageStatMap[gateMessage.MessageId]
		if messageStat == nil {
			messageStat = &GateMessageStat{MessageId: gateMessage.MessageId, MessageName: message2.MID_name[gateMessage.MessageId]}
			messageStat.SizeMin = math.MaxInt
			messageStat.StartTime = gateMessage.Time
			messageStatMap[gateMessage.MessageId] = messageStat
		}
		messageStat.Count++
		protoLength := len(gateMessage.Bytes)
		messageStat.SizeTotal += protoLength
		if protoLength > messageStat.SizeMax {
			messageStat.SizeMax = protoLength
		}
		if protoLength < messageStat.SizeMin {
			messageStat.SizeMin = protoLength
		}
		messageStat.EndTime = gateMessage.Time

		//统计序列号消息
		if gateMessage.Seq > 0 {
			messageIdPrefix := gateMessage.MessageId / 1000000
			seqStat := seqStatMap[gateMessage.Seq]
			if seqStat == nil {
				seqStat = &SeqMessage{DelayMin: math.MaxInt16}
				seqStatMap[gateMessage.Seq] = seqStat
			}
			//请求消息
			if messageIdPrefix == 3 {
				seqStat.MessageId = gateMessage.MessageId
				seqStat.RequestCount++
				seqStat.RequestTime = gateMessage.Time
			} else {
				seqStat.ResponseCount++
				if seqStat.RequestTime > 0 {
					delayTime := int(gateMessage.Time - seqStat.RequestTime)
					seqStat.RequestTime = 0
					seqStat.TotalTime += delayTime
					if delayTime > seqStat.DelayMax {
						seqStat.DelayMax = delayTime
					}
					if delayTime < seqStat.DelayMin {
						seqStat.DelayMin = delayTime
					}
				} else {
					log.Debug("没有截取到请求消息or乱序了？")
				}
			}
		}
	}

	//更新消息时长信息
	for _, seqMsg := range seqStatMap {
		messageStat := messageStatMap[seqMsg.MessageId]
		if messageStat == nil {
			log.Warn("%v 未找到消息数据", seqMsg.MessageId)
			continue
		}
		messageStat.TotalTime += seqMsg.TotalTime
		if seqMsg.DelayMin < messageStat.DelayMin {
			messageStat.DelayMin = seqMsg.DelayMin
		}
		if seqMsg.DelayMax > messageStat.DelayMax {
			messageStat.DelayMax = seqMsg.DelayMax
		}
		failCount := seqMsg.RequestCount - seqMsg.ResponseCount
		if failCount < 0 {
			failCount = 0
		}
		messageStat.FailCount += failCount
		messageStat.SuccessCount += seqMsg.ResponseCount
		if seqMsg.RequestCount > 1 {
			messageStat.RequestRepeatCount += seqMsg.RequestCount - 1
		}
		if seqMsg.ResponseCount > 1 {
			messageStat.ResponseRepeatCount += seqMsg.ResponseCount - 1
		}
	}

	// 汇总统计
	messageStats := make([]*GateMessageStat, 0, len(messageStatMap))
	for _, messageStat := range messageStatMap {
		if messageStat.SuccessCount > 0 {
			messageStat.DelayAverage = messageStat.TotalTime / messageStat.SuccessCount
		} else {
			messageStat.DelayAverage = -1
		}

		messageStat.FailRate = float32(messageStat.FailCount) / float32(messageStat.Count)
		messageStat.SizeAverage = messageStat.SizeTotal / messageStat.Count
		if messageStat.StartTime == messageStat.EndTime {
			messageStat.Rps = -1
		} else {
			messageStat.Rps = float32(messageStat.Count) / float32(messageStat.EndTime-messageStat.StartTime)
		}
		messageStats = append(messageStats, messageStat)
	}

	return messageStats
}

// 统计所有消息
func calculateAllMessageStat(socketStat *mode.SocketStat, gateMessages []*GateMessage) {
	messageStats := CalculateMessageStat(gateMessages)
	for _, stat := range messageStats {
		messageStat := GateMessageStats[stat.MessageId]
		if messageStat == nil {
			messageStat = &GateMessageStat{}
			messageStat = &GateMessageStat{MessageId: stat.MessageId, MessageName: message2.MID_name[stat.MessageId]}
			messageStat.SizeMin = math.MaxInt16
			messageStat.DelayMin = math.MaxInt16
			messageStat.StartTime = stat.StartTime
			GateMessageStats[stat.MessageId] = messageStat
		}
		earlyWarning(socketStat, stat)
		messageStat.add(stat)
	}

}

// 累加数据
func (m *GateMessageStat) add(stat *GateMessageStat) {
	m.TotalTime += stat.TotalTime
	if m.DelayMin > stat.DelayMin {
		m.DelayMin = stat.DelayMin
	}
	if m.DelayMax < stat.DelayMax {
		m.DelayMax = stat.DelayMax
	}
	m.Count += stat.Count
	m.FailCount += stat.FailCount
	m.SuccessCount += stat.SuccessCount
	m.RequestRepeatCount += stat.RequestRepeatCount
	m.ResponseRepeatCount += stat.ResponseRepeatCount
	m.SizeTotal += stat.SizeTotal
	if m.SizeMin > stat.SizeMin {
		m.SizeMin = stat.SizeMin
	}
	if m.SizeMax < stat.SizeMax {
		m.SizeMax = stat.SizeMax
	}
	if m.EndTime < stat.EndTime {
		m.EndTime = stat.EndTime
	}

}

// CalculateAverage 计算平均值
func (m *GateMessageStat) CalculateAverage() {
	if m.SuccessCount > 0 {
		m.DelayAverage = m.TotalTime / m.SuccessCount
	} else {
		m.DelayAverage = -1
	}
	m.FailRate = float32(m.FailCount) / float32(m.Count)
	m.SizeAverage = m.SizeTotal / m.Count
	if m.StartTime == m.EndTime {
		m.Rps = -1
	} else {
		m.Rps = float32(m.Count) / float32(m.EndTime-m.StartTime)
	}
}
