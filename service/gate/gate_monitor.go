package gate

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/golib/log"
	"time"
)

//1. 失败数>10&&失败率>0.1 ==> 网络监控：玩家%v ip=[] 消息=[%v=%v] 请求=%v次 失败=%v次 失败率=%v
//2. 请求数>10&&平均延迟>200 ==> 网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 平均延迟=%vms 最大延迟=%vms
//3. 请求数>100&&（重复请求>10||重复返回>10) ==> 网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 重复请求=%v 重复返回=%vms

// 预警
func earlyWarning(socketStat *mode.SocketStat, messageStat *GateMessageStat) {
	//1. 失败数>10&&失败率>0.1 ==> 网络监控：玩家%v ip=[] 消息=[%v=%v] 请求=%v次 失败=%v次 失败率=%v
	//客户端断线重连有问题，断线重连会出现几次创建socket不发消息，然后只有一条消息没收到，在这期间玩家一直点击，序列号递增，可能一次性发几十条相同消息，因此失败率较高
	if messageStat.FailCount > 100 && messageStat.FailRate > 0.2 {
		sentry.CaptureMessage(fmt.Sprintf("网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 失败=%v次 失败率=%v",
			socketStat.MessageStat.AppStat.(*GateStat).PlayerId, socketStat.Connection.String(), messageStat.MessageId, messageStat.MessageName,
			messageStat.Count, messageStat.FailCount, messageStat.FailRate))
		//2. 请求数>100&&平均延迟>100 ==> 网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 平均延迟=%vms 最大延迟=%vms
	} else if messageStat.Count > 30 && messageStat.DelayAverage > 1000 {
		sentry.CaptureMessage(fmt.Sprintf("网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 平均延迟=%vms 最大延迟=%vms",
			socketStat.MessageStat.AppStat.(*GateStat).PlayerId, socketStat.Connection.String(), messageStat.MessageId, messageStat.MessageName,
			messageStat.Count, messageStat.DelayAverage, messageStat.DelayMax))
		//3. 请求数>100&&（重复请求>10||重复返回>10) ==> 网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 重复请求=%v 重复返回=%vms
	} else if messageStat.Count > 100 && (messageStat.RequestRepeatCount > 10 || messageStat.ResponseRepeatCount > 10) {
		sentry.CaptureMessage(fmt.Sprintf("网络监控：玩家%v ip=[%v] 消息=[%v=%v] 请求=%v次 重复请求=%v 重复返回=%v",
			socketStat.MessageStat.AppStat.(*GateStat).PlayerId, socketStat.Connection.String(), messageStat.MessageId, messageStat.MessageName,
			messageStat.Count, messageStat.RequestRepeatCount, messageStat.ResponseRepeatCount))
	}
}

// Start 启动配置sentry
func startSentry() {
	// 只有线上拥有sentry
	if config.ApplicationConfigInstance.Profile != "online" {
		return
	}
	err := sentry.Init(sentry.ClientOptions{
		// Either set your DSN here or set the SENTRY_DSN environment variable.
		//暂时写死，所有微服务公用一个
		Dsn: "http://05ac7f7f58a4209fd122f7a89dbfb@localhost:9000/7",
		// Either set environment and release here or set the SENTRY_ENVIRONMENT
		// and SENTRY_RELEASE environment variables.
		Environment: config.ApplicationConfigInstance.Profile,
		Release:     time.Now().Format(" 2006-01-02"),
		// Enable printing of SDK debug messages.
		// Useful when getting started or trying to figure something out.
		Debug: true,
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		TracesSampleRate: 0.1,
		ServerName:       "net-service",
	})
	if err != nil {
		log.Error("sentry.Init: %s", err)
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)
}
