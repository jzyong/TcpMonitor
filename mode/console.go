package mode

const ConsoleCacheCount = 1024 * 10

// ConsoleLogType 控制台日志类型
type ConsoleLogType int

const (
	SocketCreate ConsoleLogType = 0x0001 //socket创建
	SocketClose  ConsoleLogType = 0x0002 //socket关闭
	Message      ConsoleLogType = 0x0004 //消息详细信息
)

// ConsoleLog 控制台输出
type ConsoleLog struct {
	Index  int64          `json:"index"`  //索引
	Id     string         `json:"id"`     //ID
	Time   string         `json:"time"`   //时间
	Type   ConsoleLogType `json:"type"`   //类型
	Flow   string         `json:"flow"`   //流向
	Log1   string         `json:"log1"`   //输出日志1
	Log2   string         `json:"log2"`   //输出日志2
	Object any            `json:"object"` //对象
}
