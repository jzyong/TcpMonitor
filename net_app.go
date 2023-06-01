package main

import (
	"flag"
	"github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/service/gate"
	"github.com/jzyong/TcpMonitor/service/http"
	"github.com/jzyong/TcpMonitor/web/router"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	log2 "log"
	"runtime"
)

var (
	runMode = flag.String("m", "server", "运行模式：\r\n"+
		"server 服务器 \r\n"+
		"gate-cli 打印slots网关详细信息（包括消息装配）\r\n"+
		"http 截取HTTP消息\r\n"+
		"device 查看网卡名称列表\r\n")
	device     = flag.String("device", "\\Device\\NPF_Loopback", "网卡名称")
	filter     = flag.String("filter", "tcp and port 7010", "BPFFilter过滤表达式")
	configPath = flag.String("config", "D:\\Go\\TcpMonitor\\config\\ApplicationConfig_develop_gate.json", "配置文件加载路径")
)

// ModuleManager 模块管理
type ModuleManager struct {
	*util.DefaultModuleManager
	NetManager     *manager.NetManager
	StatManager    *manager.StatManager
	DataManager    *manager.DataManager
	SystemManager  *manager.SystemManager
	ConsoleManager *manager.ConsoleManager
}

// Init 初始化模块
func (m *ModuleManager) Init() error {
	m.NetManager = m.AppendModule(manager.GetNetManager()).(*manager.NetManager)
	m.StatManager = m.AppendModule(manager.GetStatManager()).(*manager.StatManager)
	m.DataManager = m.AppendModule(manager.GetDataManager()).(*manager.DataManager)
	m.SystemManager = m.AppendModule(manager.GetSystemManager()).(*manager.SystemManager)
	m.ConsoleManager = m.AppendModule(manager.GetConsoleManager()).(*manager.ConsoleManager)
	return m.DefaultModuleManager.Init()
}

var m = &ModuleManager{
	DefaultModuleManager: util.NewDefaultModuleManager(),
}

// 后台统计入口类
func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	if flagOperate(*runMode) {
		return
	}

	initConfigAndLog()

	log.Info("start net service ......")

	var err error
	err = m.Init()
	if err != nil {
		log.Error("net service start error: %s", err.Error())
		return
	}
	m.Run()
	serverRun()
	util.WaitForTerminate()
	m.Stop()

	util.WaitForTerminate()
}

// 初始化项目配置和日志
func initConfigAndLog() {
	//1.配置文件路径
	//configPath := flag.String("config", "D:\\Go\\slots-service\\net-service\\config\\ApplicationConfig_jzy_manage.json", "配置文件加载路径")
	//flag.Parse()
	config.FilePath = *configPath
	config.ApplicationConfigInstance.Reload()

	//2.关闭debug
	if "DEBUG" != config.ApplicationConfigInstance.LogLevel {
		log.CloseDebug()
	}
	log.SetLogFile("log", "net")

}

// 控制台操作 true只控制台运行
func flagOperate(operateType string) bool {

	log2.Printf("运行模式：%v", operateType)
	switch operateType {
	case "gate-cli":
		gate.GateCliStart(*device, *filter)
		return true
	case "http":
		http.CliStart(*device, *filter)
		return true
	case "device":
		manager.PrintAllDevices()
		return true
	}

	return false
}

// 根据配置启动服务器模式
func serverRun() {
	gate.ServerStart()
	router.StartWeb()
}
