package manager

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NetManager  基础入口
type NetManager struct {
	util.DefaultModule
}

var netManager *NetManager
var netSingletonOnce sync.Once

func GetNetManager() *NetManager {
	netSingletonOnce.Do(func() {
		netManager = &NetManager{}
	})
	return netManager
}

func (m *NetManager) Init() error {
	log.Info("NetManager init start......")

	//go m.StartHttp()
	log.Info("NetManager started ......")

	return nil
}

func (m *NetManager) Run() {
	//log.Info("临时测试加载大厅服务")
}

func (m *NetManager) Stop() {
}

// 解析网络端口,windows本地目标端口为7010(ups-onlinet)
func parsePort(s string) uint16 {
	idx := strings.Index(s, "(")
	if idx == -1 {
		i, _ := strconv.Atoi(s)
		return uint16(i)
	}

	i, _ := strconv.Atoi(s[:idx])
	return uint16(i)
}

// ListenNetCard 需要新启动go routine
func (m *NetManager) ListenNetCard(assembler *tcpassembly.Assembler) {
	config := config2.ApplicationConfigInstance
	if handle, err := pcap.OpenLive(config.Device, config.SnapLen, true, pcap.BlockForever); err != nil {
		panic(err)
	} else if err := handle.SetBPFFilter(config.BPFFilter); err != nil { // optional
		panic(err)
	} else {

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		packetSource.NoCopy = true
		packets := packetSource.Packets()
		ticker := time.Tick(time.Second * 5)
		for {
			select {
			case packet := <-packets:
				// A nil packet indicates the end of a pcap file.
				if packet == nil {
					log.Warn("接收到空 packet")
					continue
				}
				if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
					log.Debug("Unusable packet")
					continue
				}
				GetStatManager().packets <- packet
				tcp := packet.TransportLayer().(*layers.TCP)
				//	log.Info("源端口：%v 目标端口：%v SYN=%v PSH=%v FIN=%v", parsePort(tcp.SrcPort.String()), parsePort(tcp.DstPort.String()), tcp.SYN, tcp.PSH, tcp.FIN)
				assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

			case <-ticker:
				// Every minute, flush connections that haven't seen activity in the past 2 minutes.
				//assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
				assembler.FlushOlderThan(time.Now().Add(time.Second * -10))

			}
		}
	}
}

// PrintAllDevices 输出所有网卡设备
func PrintAllDevices() {
	if devices, err := pcap.FindAllDevs(); err != nil {
		log.Warn("获取网卡失败：%v", err)
	} else {
		for _, device := range devices {
			log.Info("网卡：%v ==> %v", device.Name, device.Description)
			for _, address := range device.Addresses {
				log.Info("IP:%v", address)
			}
		}
	}
}
