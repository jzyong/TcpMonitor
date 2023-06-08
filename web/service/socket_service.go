package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/snappy"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/TcpMonitor/service/gate"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GetSocketList 分页查询Socket 列表
func GetSocketList(start, count int, search, sort, order string) (socketStats []*mode.SocketStat, totalCount int64) {
	collection := manager.GetDataManager().GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := bson.M{}
	//暂时支持源ip和玩家ID模糊查询
	if len(search) > 0 {
		regex := bson.M{"$regex": search, "$options": "i"}
		//query = bson.M{"connection.local.ip": regex}
		query = bson.M{
			"$or": []bson.M{
				bson.M{"connection.local.ip": regex},
				bson.M{"messageStat.appStat.playerId": regex},
			},
		}
	}
	opts := options.Find()
	if len(sort) > 0 {
		o := -1
		if order == "asc" {
			o = 1
		}
		opts.SetSort(bson.D{{util.FirstLower(sort), o}}) // 首字母小写
		//默认根据创建时间倒序
	} else {
		opts.SetSort(bson.D{{"beginTime", -1}})
	}

	opts.SetLimit(int64(count))
	opts.SetSkip(int64(start))
	cur, err := collection.Find(ctx, query, opts)
	socketStats = make([]*mode.SocketStat, 0, 1000)
	if err != nil {
		log.Error("%v", err)
		return socketStats, 0
	}

	for cur.Next(ctx) {
		var result = &mode.SocketStat{}
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal("%v", err)
		}
		socketStats = append(socketStats, result)
	}
	totalCount, err = collection.CountDocuments(ctx, query)
	if err != nil {
		log.Warn("查询数目错误:%v", err)
	}

	return socketStats, totalCount
}

// BuildSocketInfos 转换为客户端显示的信息
func BuildSocketInfos(socketStats []*mode.SocketStat) []*SocketInfo {
	infos := make([]*SocketInfo, 0, len(socketStats))
	if socketStats == nil {
		return infos
	}
	for _, stat := range socketStats {
		reconnect := false
		var timeOutCount int32 = 0
		var executeAverageTime int32 = 0
		var executeTime int32 = 0
		var seqCount int32 = 0
		var messageRetransmissionRate float32 = 0
		var playerId string
		if stat.MessageStat.AppStat != nil {
			gateStat := stat.MessageStat.AppStat.(primitive.D).Map()
			if r, ok := gateStat["reconnect"]; ok {
				reconnect = r.(bool)
			}
			if r, ok := gateStat["playerId"]; ok {
				playerId = r.(string)
			}

			timeOutCount = gateStat["timeOutCount"].(int32)
			executeTime = gateStat["seqExecuteTime"].(int32)
			seqCount = gateStat["seqCount"].(int32)

			if seqCount > 0 {
				executeAverageTime = executeTime / seqCount
				messageRetransmissionRate = float32(timeOutCount) / float32(seqCount)
			}

		}

		info := &SocketInfo{
			Id:                        stat.Id,
			Connection:                stat.Connection.String(),
			BeginTime:                 time.UnixMilli(stat.BeginTime).Format("2006-01-02 15:04:05"),
			EndTime:                   time.UnixMilli(stat.EndTime).Format("2006-01-02 15:04:05"),
			CloseType:                 closeType(stat),
			PacketCount:               packetCount(stat),
			ByteSize:                  byteSize(stat),
			Rps:                       rps(stat),
			MessageMaxSize:            stat.MessageStat.MaxBytes,
			Reconnect:                 reconnect,
			TimeOutCount:              timeOutCount,
			ExecuteAverageTime:        executeAverageTime,
			SYN:                       stat.SYN,
			ExecuteTime:               executeTime,
			SeqRequestCount:           seqCount,
			Duration:                  stat.DurationSecond(),
			PacketRetransmissionRate:  fmt.Sprintf("%.2f", stat.PacketStat.RetransmissionRate()*100),
			MessageRetransmissionRate: fmt.Sprintf("%.2f", messageRetransmissionRate*100),
			PlayerId:                  playerId,
			CountryCity:               manager.GetDataManager().GetCountyCity(stat.Connection.Local.Ip),
			Rtt:                       stat.Rtt,
			MinTTL:                    stat.MinTTL,
			MinWindowSize:             stat.MinWindowSize,
		}
		infos = append(infos, info)

	}
	return infos
}

// 关闭类型
func closeType(stat *mode.SocketStat) string {
	if stat.FINClient {
		return "Client(Fin)"
	} else if stat.RSTClient {
		return "Client(Rst)"
	} else if stat.FINServer {
		return "Server(Fin)"
	} else if stat.RSTServer {
		return "server(Rst)"
	} else {
		return "TimeOut"
	}
}

// 包个数 上行消息数/包数/重传包数  下行消息数/包数/重传包数
func packetCount(stat *mode.SocketStat) string {
	p := stat.PacketStat
	return fmt.Sprintf("<span style=\"display: inline-block; width: 100px;\">%v/%v/%v</span>%v/%v/%v", stat.MessageStat.UploadCount, p.UploadPackets, p.UploadRetransmissionCount,
		stat.MessageStat.DownloadCount, p.DownloadPackets, p.DownRetransmissionCount)
}

// 字节大小（B，KB、MB） 上行消息字节数/包字节数  下行消息字节数/包字节数
func byteSize(stat *mode.SocketStat) string {
	return fmt.Sprintf("<span style=\"display: inline-block; width: 110px;\">%v/%v</span>%v/%v", util.ByteConvertString(float32(stat.MessageStat.UploadBytes)),
		util.ByteConvertString(float32(stat.PacketStat.UploadBytes)), util.ByteConvertString(float32(stat.MessageStat.DownloadBytes)),
		util.ByteConvertString(float32(stat.PacketStat.DownloadBytes)))
}

// rps 上行/下行
func rps(stat *mode.SocketStat) string {
	executeTime := float32((stat.ActiveTime - stat.BeginTime) / 1000)
	if executeTime < 0.1 {
		return fmt.Sprintf("0/0")
	}
	return fmt.Sprintf("%.2f/%.2f", float32(stat.MessageStat.UploadCount)/executeTime, float32(stat.MessageStat.DownloadCount)/executeTime)
}

// SocketInfo web展示的socket 列表信息
type SocketInfo struct {
	Id                 string //唯一标识ID
	Connection         string //连接地址
	BeginTime          string //开始时间
	EndTime            string //结束时间
	CloseType          string //关闭类型（客户端，服务器，超时）
	PacketCount        string //包个数 上行消息数/包数/重传包数  下行消息数/包数/重传包数
	ByteSize           string //字节大小（B，KB、MB） 上行消息字节数/包字节数  下行消息字节数/包字节数
	Rps                string //rps 上行/下行
	Reconnect          bool   //是否未重连
	ExecuteAverageTime int32  //平均执行时间 ms
	PlayerId           string //玩家id
	Rtt                uint32 //往返时间

	//下面为详细页面展示
	Duration                  int32  //持续时间s
	SYN                       bool   //是否截取到创建
	ExecuteTime               int32  //总执行时间
	SeqRequestCount           int32  //序列号请求消息数
	MessageMaxSize            uint32 //消息最大长度
	TimeOutCount              int32  //超时数
	PacketRetransmissionRate  string //包重传率
	MessageRetransmissionRate string //消息重传率
	CountryCity               string //国家城市
	MinTTL                    uint8  //最小TTL
	MinWindowSize             uint16 //最小窗口大小
}

// GetGateMessages 获取网关消息
func GetGateMessages(id string) (*gate.GateMessages, error) {
	collection := manager.GetDataManager().GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("message")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := bson.M{"_id": id}
	result := collection.FindOne(ctx, query)
	if result.Err() != nil {
		return nil, result.Err()
	}
	messages := &gate.GateMessages{}
	err := result.Decode(&messages)
	if err != nil {
		return nil, err
	}

	//解压缩 ,使用Byte替换messages
	if messages.Bytes != nil && len(messages.Bytes) > 0 {
		decodeBuf, err := snappy.Decode(nil, messages.Bytes)
		if err != nil {
			log.Warn("解byte错误:%v", err)
			return nil, err
		}
		messages2 := make([]*gate.GateMessage, 0, 500)
		err = json.Unmarshal(decodeBuf, &messages2)
		if err != nil {
			log.Warn("解json错误:%v", err)
			return nil, err
		}
		// 对消息进行排序，assembly时不能保证有序性性，根据截取包的时间进行排序
		sort.Slice(messages2, func(i, j int) bool {
			return messages2[i].Time < messages2[j].Time
		})
		messages.Message = messages2
	}
	return messages, nil
}

// GateMessageStatPageAndSort 对消息统计进行排序分页
func GateMessageStatPageAndSort(messageStats []*gate.GateMessageStat, search, sortStr, order string, start, length int) ([]*gate.GateMessageStat, int) {
	if len(search) > 0 {
		messageStats2 := make([]*gate.GateMessageStat, 0, len(messageStats))
		for _, stat := range messageStats {
			if strings.Contains(stat.MessageName, search) || strings.Contains(strconv.Itoa(int(stat.MessageId)), search) {
				messageStats2 = append(messageStats2, stat)
			}
		}
		messageStats = messageStats2
	}
	var sortFunc func(c1, c2 *gate.GateMessageStat) bool = nil
	switch sortStr {
	case "messageId":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.MessageId < c2.MessageId
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.MessageId > c2.MessageId
			}
		}
	case "messageName":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				result := strings.Compare(c1.MessageName, c2.MessageName)
				if result < 0 {
					return true
				}
				return false
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				result := strings.Compare(c1.MessageName, c2.MessageName)
				if result > 0 {
					return true
				}
				return false
			}
		}
	case "totalTime":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.TotalTime < c2.TotalTime
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.TotalTime > c2.TotalTime
			}
		}
	case "delayAverage":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.DelayAverage < c2.DelayAverage
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.DelayAverage > c2.DelayAverage
			}
		}
	case "delayMax":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.DelayMax < c2.DelayMax
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.DelayMax > c2.DelayMax
			}
		}
	case "count":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.Count < c2.Count
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.Count > c2.Count
			}
		}
	case "failCount":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.FailCount < c2.FailCount
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.FailCount > c2.FailCount
			}
		}

	case "failRate":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.FailRate < c2.FailRate
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.FailRate > c2.FailRate
			}
		}
	case "requestRepeatCount":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.RequestRepeatCount < c2.RequestRepeatCount
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.RequestRepeatCount > c2.RequestRepeatCount
			}
		}
	case "responseRepeatCount":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.ResponseRepeatCount < c2.ResponseRepeatCount
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.ResponseRepeatCount > c2.ResponseRepeatCount
			}
		}
	case "sizeTotal":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.SizeTotal < c2.SizeTotal
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.SizeTotal > c2.SizeTotal
			}
		}
	case "sizeAverage":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.SizeAverage < c2.SizeAverage
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.SizeAverage > c2.SizeAverage
			}
		}
	case "sizeMax":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.SizeMax < c2.SizeMax
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.SizeMax > c2.SizeMax
			}
		}
	case "startTime":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.StartTime < c2.StartTime
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.StartTime > c2.StartTime
			}
		}
	case "endTime":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.EndTime < c2.EndTime
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.EndTime > c2.EndTime
			}
		}
	case "rps":
		if order == "asc" {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.Rps < c2.Rps
			}
		} else {
			sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
				return c1.Rps > c2.Rps
			}
		}

	default:
		//延迟降序排序
		sortFunc = func(c1, c2 *gate.GateMessageStat) bool {
			return c1.DelayAverage > c2.DelayAverage
		}

	}

	sort.Slice(messageStats, func(i, j int) bool {
		return sortFunc(messageStats[i], messageStats[j])
	})

	end := start + length
	if end > len(messageStats) {
		end = len(messageStats)
	}
	returnStats := messageStats[start:end]

	return returnStats, len(messageStats)
}
