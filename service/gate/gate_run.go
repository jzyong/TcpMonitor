package gate

import (
	"github.com/google/gopacket/tcpassembly"
	"github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
)

// GateCliStart Gate控制台输出
func GateCliStart(device, BPFFilter string) {
	log.SetLogFile("../log", "net")
	config := config.ApplicationConfigInstance
	config.Device = device
	config.BPFFilter = BPFFilter
	config.UnmarshalPacket = true
	go manager.GetStatManager().ProcessStats()
	ServerStart()
	util.WaitForTerminate()
}

// ServerStart 服务器启动
func ServerStart() {
	streamFactory := &GateStreamFactory{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	manager.GetStatManager().AppStatFun = processStat
	manager.GetDataManager().CreateMessageFun = newMessages
	starIndexLineStat()
	go manager.GetNetManager().ListenNetCard(assembler)
	go startSentry() //预警监控
}
