package controller

import (
	"fmt"
	"github.com/jzyong/TcpMonitor/service/gate"
	"github.com/jzyong/TcpMonitor/service/gate/message"
	"github.com/jzyong/TcpMonitor/web/service"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"google.golang.org/protobuf/proto"
	"strings"
	"time"
)

// SocketController socket信息
type SocketController struct {
	BaseController
}

// List 列表显示
func (m *SocketController) List() {
	if m.Ctx.Request.Method == "GET" {
		m.displayActive("socket/list.html", "socket")
		return
	}

	start, length := m.GetAjaxPageParams()
	log.Debug(" 列表请求 start=%v length=%v search=%v sort=%v order=%v", start, length, m.getEscapeString("search"),
		m.getEscapeString("sort"), m.getEscapeString("order"))
	search := m.getEscapeString("search")
	sort := m.getEscapeString("sort")
	order := m.getEscapeString("order")

	socketStats, totalCount := service.GetSocketList(start, length, search, sort, order)

	m.AjaxTable(service.BuildSocketInfos(socketStats), len(socketStats), int(totalCount), nil)

}

// Message 获取socket对应的消息详细信息
func (m *SocketController) Message() {
	id := m.getEscapeString("id")
	log.Info("获取消息：%v", id)

	messages, err := service.GetGateMessages(id)
	if err != nil {
		m.Data["json"] = fmt.Sprintf("获取数据异常：%v", err)
		m.ServeJSON()
		m.StopRun()
		return
	}

	var build strings.Builder
	for i, gateMessage := range messages.Message {
		if gateMessage.MessageId < 1 {
			continue
		}
		color := "green"
		if gateMessage.MessageId/1000000 == 5 {
			color = "blue"
		}
		//<strong style="color: #3dc7ab">ss</strong>
		header := fmt.Sprintf("%v\t%v\t<strong style=\"color: %v\">Id=%v\tSeq=%v\tAck=%v\tName=%v</strong>", i, time.UnixMilli(gateMessage.Time).Format("15:04:05"),
			color, gateMessage.MessageId, gateMessage.Seq, gateMessage.Ack, message.MID_name[gateMessage.MessageId])

		build.WriteString(header)
		build.WriteString("\r\n")
		if len(gateMessage.Bytes) > 0 {
			protoMessage := gate.MessageDecoder[gateMessage.MessageId]
			if protoMessage != nil {
				proto.Unmarshal(gateMessage.Bytes, protoMessage)
				//build.WriteString(fmt.Sprintf("\t%+v", protoMessage))
				build.WriteString(fmt.Sprintf("\t\t\t%v", util.ToString(protoMessage))) // json 会报错？
				//build.WriteString(fmt.Sprintf("\t\t\t%v", util.ToStringIndent(protoMessage))) //
			} else {
				build.WriteString("\t\t\t<strong style=\"color: red\">proto协议未注册，请添加</strong>")
			}
			build.WriteString("\r\n")
		}
	}

	m.Data["json"] = build.String()
	m.ServeJSON()
	m.StopRun()
}

// MessageList 消息列表显示
func (m *SocketController) MessageList() {

	start, length := m.GetAjaxPageParams()
	search := m.getEscapeString("search")
	sort := m.getEscapeString("sort")
	order := m.getEscapeString("order")
	id := m.getEscapeString("id")
	log.Info(" 列表请求 start=%v length=%v search=%v sort=%v order=%v id=%v", start, length, search,
		sort, order, id)

	messages, err := service.GetGateMessages(id)
	if err != nil {
		m.Data["json"] = fmt.Sprintf("获取数据异常：%v", err)
		m.ServeJSON()
		m.StopRun()
		return
	}

	statMessages := gate.CalculateMessageStat(messages.Message)
	// 排序，查询，分页等
	showMessageStats, totalCount := service.GateMessageStatPageAndSort(statMessages, search, sort, order, start, length)
	m.AjaxTable(showMessageStats, len(showMessageStats), totalCount, nil)
}
