package controller

import (
	"fmt"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/TcpMonitor/service/gate"
	"github.com/jzyong/TcpMonitor/service/gate/message"
	"github.com/jzyong/golib/log"
	"google.golang.org/protobuf/proto"
	"strings"
)

// ConsoleController 控制台信息
type ConsoleController struct {
	BaseController
}

// List 列表显示
func (m *ConsoleController) List() {
	if m.Ctx.Request.Method == "GET" {
		m.displayActive("console/list.html", "console")
		// 充值设置
		m.SetSession("consoleIndex", int64(0))
		m.SetSession("consoleFilter", "")
		m.SetSession("consoleType", 0)
		return
	}

	index := m.GetSession("consoleIndex").(int64)

	if index < 1 { //第一次打开页面，从缓存中获取历史1024条
		index = manager.GetConsoleManager().LogIndex - 512
	}
	logs := make([]*mode.ConsoleLog, 0, 10)
	newIndex := index
	if index < manager.GetConsoleManager().LogIndex {
		consoleLogs := manager.GetConsoleManager().Logs
		filterStr := m.GetSession("consoleFilter").(string)
		showType := m.GetSession("consoleType").(int) //3显示socket创建和关闭；0默认基础显示；4显示详细反序列化消息
		for _, consoleLog := range consoleLogs {
			if consoleLog.Index > index {
				//显示类型筛选 只显示socket创建和关闭
				if showType == 3 && consoleLog.Type == mode.Message {
					continue
				}

				//消息类型从新生成内容，
				if consoleLog.Type == mode.Message && len(consoleLog.Log1) < 1 {
					gateMessage := consoleLog.Object.(*gate.GateMessage)
					if len(consoleLog.Log1) < 1 {
						logStr1 := fmt.Sprintf("Seq=%v Ack=%v MID=%v Name=%v Len=%v PlayerId=%v", gateMessage.Seq, gateMessage.Ack, gateMessage.MessageId, message.MID(gateMessage.MessageId), gateMessage.Length, consoleLog.Id)
						consoleLog.Log1 = logStr1
					}
					//反序列化消息
					if showType == 4 && len(consoleLog.Log2) < 1 && gateMessage.Bytes != nil {
						msg := gate.MessageDecoder[gateMessage.MessageId]
						if msg != nil {
							proto.Unmarshal(gateMessage.Bytes, msg)
							consoleLog.Log2 = fmt.Sprintf("%+v", msg)
						} else {
							consoleLog.Log2 = fmt.Sprintf("消息 %v 未注册，请更新protobuf并添加", gateMessage.MessageId)
						}
					}
				}
				//过滤内容
				if len(filterStr) > 0 && !strings.Contains(consoleLog.Log1, filterStr) && !strings.Contains(consoleLog.Flow, filterStr) {
					continue
				}
				logs = append(logs, consoleLog)
				newIndex = consoleLog.Index
			}
		}
	}
	m.SetSession("consoleIndex", newIndex)
	m.Data["json"] = logs
	m.ServeJSON()
	m.StopRun()
}

// Filter 过滤器设置
func (m *ConsoleController) Filter() {
	consoleFilter := m.getEscapeString("filter")
	consoleType := m.GetIntNoErr("type", 0)
	log.Info("收到console数据:filter=%s  type=%d", consoleFilter, consoleType)
	m.SetSession("consoleFilter", consoleFilter)
	m.SetSession("consoleType", consoleType)
	m.Data["json"] = "过滤器设置成功"
	m.ServeJSON()
	m.StopRun()
}
