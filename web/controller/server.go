package controller

import (
	"github.com/jzyong/TcpMonitor/manager"
	"strconv"
)

// ServerController 服务器信息
type ServerController struct {
	BaseController
}

// List 列表显示
func (m *ServerController) List() {
	data := make(map[string]interface{})
	serverStatus := manager.GetSystemManager().ServerStatus
	var fg int
	if len(serverStatus) >= 10 {
		fg = len(serverStatus) / 10
		for i := 0; i <= 9; i++ {
			data["sys"+strconv.Itoa(i+1)] = serverStatus[i*fg]
		}
	}
	m.Data["data"] = data
	m.displayActive("server/list.html", "server")
}
