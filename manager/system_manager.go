package manager

import (
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"math"
	"sync"
	"time"
)

// SystemManager  系统工具，CPU、内存、网络等数据获取
type SystemManager struct {
	util.DefaultModule
	ServerStatus []map[string]interface{}
}

var systemManager *SystemManager
var systemSingletonOnce sync.Once

func GetSystemManager() *SystemManager {
	systemSingletonOnce.Do(func() {
		systemManager = &SystemManager{
			ServerStatus: make([]map[string]interface{}, 0, 1500),
		}
	})
	return systemManager
}

func (s *SystemManager) Init() error {
	log.Info("SystemManager init start......")

	go s.getSeverStatus()
	log.Info("SystemManager started ......")

	return nil
}

func (s *SystemManager) Run() {
}

func (s *SystemManager) Stop() {
}

// 获取CPU 、内存、网络等信息
func (s *SystemManager) getSeverStatus() {
	for {
		if len(s.ServerStatus) < 10 {
			time.Sleep(time.Second)
		} else {
			time.Sleep(time.Minute)
		}
		cpuPercet, _ := cpu.Percent(0, true)
		var cpuAll float64
		for _, v := range cpuPercet {
			cpuAll += v
		}
		m := make(map[string]interface{})
		loads, _ := load.Avg()
		m["load1"] = loads.Load1
		m["load5"] = loads.Load5
		m["load15"] = loads.Load15
		m["cpu"] = math.Round(cpuAll / float64(len(cpuPercet)))
		swap, _ := mem.SwapMemory()
		m["swap_mem"] = math.Round(swap.UsedPercent)
		vir, _ := mem.VirtualMemory()
		m["virtual_mem"] = math.Round(vir.UsedPercent)
		conn, _ := net.ProtoCounters(nil)
		io1, _ := net.IOCounters(false)
		time.Sleep(time.Millisecond * 500)
		io2, _ := net.IOCounters(false)
		if len(io2) > 0 && len(io1) > 0 {
			m["io_send"] = (io2[0].BytesSent - io1[0].BytesSent) * 2
			m["io_recv"] = (io2[0].BytesRecv - io1[0].BytesRecv) * 2
		}
		t := time.Now()
		//m["time"] = strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + ":" + strconv.Itoa(t.Second())
		m["time"] = t.Format("15:04:05")

		for _, v := range conn {
			m[v.Protocol] = v.Stats["CurrEstab"]
		}
		if len(s.ServerStatus) >= 1440 {
			s.ServerStatus = s.ServerStatus[1:]
		}
		s.ServerStatus = append(s.ServerStatus, m)

	}
}
