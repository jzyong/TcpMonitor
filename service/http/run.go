package http

import (
	"github.com/google/gopacket/tcpassembly"
	"github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
)

// CliStart 启动HTTP
func CliStart(device, BPFFilter string) {
	log.SetLogFile("../log", "net-service")
	config := config.ApplicationConfigInstance
	config.Device = device
	config.BPFFilter = BPFFilter
	streamFactory := &HttpStreamFactory{}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	go manager.GetNetManager().ListenNetCard(assembler)
	util.WaitForTerminate()
}
