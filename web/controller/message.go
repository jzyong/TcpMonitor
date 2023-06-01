package controller

import (
	"github.com/jzyong/TcpMonitor/service/gate"
	"github.com/jzyong/TcpMonitor/web/service"
	"github.com/jzyong/golib/log"
)

// MessageController 消息
type MessageController struct {
	BaseController
}

// List 列表显示
func (m *MessageController) List() {
	if m.Ctx.Request.Method == "GET" {
		m.displayActive("message/list.html", "message")
		return
	}

	start, length := m.GetAjaxPageParams()
	search := m.getEscapeString("search")
	sort := m.getEscapeString("sort")
	order := m.getEscapeString("order")
	id := m.getEscapeString("id")
	log.Info(" 列表请求 start=%v length=%v search=%v sort=%v order=%v id=%v", start, length, search,
		sort, order, id)

	gateMessageStats := make([]*gate.GateMessageStat, 0, len(gate.GateMessageStats))
	for _, stat := range gate.GateMessageStats {
		stat.CalculateAverage()
		gateMessageStats = append(gateMessageStats, stat)
	}

	// 排序，查询，分页等
	showMessageStats, totalCount := service.GateMessageStatPageAndSort(gateMessageStats, search, sort, order, start, length)
	m.AjaxTable(showMessageStats, len(showMessageStats), totalCount, nil)
}
