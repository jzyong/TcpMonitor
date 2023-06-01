package config

import (
	"encoding/json"
	"github.com/jzyong/golib/log"
	"os"
)

// ApplicationConfigInstance 配置
var ApplicationConfigInstance *ApplicationConfig

// FilePath 配置文件路径
var FilePath string

// ApplicationConfig 配置
type ApplicationConfig struct {
	Id              int32    `json:"id"`              //服务器ID
	DatabaseUrl     string   `json:"databaseUrl"`     //MongoDB URL
	DatabaseName    string   `json:"databaseName"`    //MongoDB name
	Profile         string   `json:"profile"`         //个性化配置
	LogLevel        string   `json:"logLevel"`        //日志级别
	BPFFilter       string   `json:"BPFFilter"`       // BPFFilter is the string pcap filter with the BPF syntax eg. "tcp and port 80"
	SnapLen         int32    `json:"snapLen"`         //消息截取长度
	Device          string   `json:"device"`          //网卡设备 window本地需要使用 Loopback
	UnmarshalPacket bool     `json:"unmarshalPacket"` //解析消息包
	ServerPort      []uint16 `json:"serverPort"`      //监听的服务器断开，拥有判断连接
	WebAccount      string   `json:"webAccount"`      //网页账号
	WebPassword     string   `json:"webPassword"`     //网页密码
	WebUrl          string   `json:"webUrl"`          //web地址
	ServerIp        string   `json:"serverIp"`        //服务器ip，拥有在地图上展示位置
}

func init() {
	ApplicationConfigInstance = &ApplicationConfig{
		Id:       1,
		LogLevel: "DEBUG",
		Profile:  "develop",
	}
	//ApplicationConfigInstance.Reload()
}

// PathExists 判断一个文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Reload 读取用户的配置文件
func (statsConfig *ApplicationConfig) Reload() {
	if confFileExists, _ := PathExists(FilePath); confFileExists != true {
		//fmt.Println("Config File ", g.ConfFilePath , " is not exist!!")
		log.Warn("config file ", FilePath, "not find, use default config")
		return
	}
	data, err := os.ReadFile(FilePath)
	if err != nil {
		panic(err)
	}
	//将json数据解析到struct中
	err = json.Unmarshal(data, statsConfig)
	if err != nil {
		log.Error("%v", err)
		panic(err)
	}
}
