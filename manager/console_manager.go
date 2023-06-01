package manager

import (
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"sync"
	"time"
)

// ConsoleManager  控制台输出
type ConsoleManager struct {
	util.DefaultModule
	Logs     []*mode.ConsoleLog //日志
	LogIndex int64              // 日志索引
	lock     sync.RWMutex
}

// AppConsoleeFun 自定义统计函数

var consoleManager *ConsoleManager
var consoleSingletonOnce sync.Once

func GetConsoleManager() *ConsoleManager {
	consoleSingletonOnce.Do(func() {
		consoleManager = &ConsoleManager{
			Logs: make([]*mode.ConsoleLog, 0, mode.ConsoleCacheCount),
		}
	})
	return consoleManager
}

func (m *ConsoleManager) Init() error {
	log.Info("ConsoleManager init start......")

	log.Info("ConsoleManager started ......")

	return nil
}

func (m *ConsoleManager) Run() {
}

func (m *ConsoleManager) Stop() {
}

// AddConsoleLog 添加console日志
func (m *ConsoleManager) AddConsoleLog(logType mode.ConsoleLogType, flow string, log string, id string, object any) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.LogIndex++
	consoleLog := &mode.ConsoleLog{Index: m.LogIndex, Type: logType}
	consoleLog.Time = time.Now().Format("2006-01-02 15:04:05")
	consoleLog.Flow = flow
	consoleLog.Log1 = log
	consoleLog.Id = id
	if len(m.Logs) >= mode.ConsoleCacheCount {
		m.Logs = m.Logs[1:]
	}
	if object != nil {
		consoleLog.Object = object
	}

	m.Logs = append(m.Logs, consoleLog)

}
