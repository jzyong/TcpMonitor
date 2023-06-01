package controller

import (
	"context"
	"fmt"
	"github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/manager"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"github.com/oschwald/geoip2-golang"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math"
	"sort"
	"strings"
	"time"
)

// CountryController 国家
// 按照国家（type=0）和城市（type=1）进行分类统计
type CountryController struct {
	BaseController
}

//======================================================================================================================
//List 列表展示页面

// List 列表显示
func (m *CountryController) List() {
	requestType := m.GetIntNoErr("type", 0)
	menu := "country"
	if requestType == 1 {
		menu = "city"
	}
	log.Info("请求类型：%v", requestType)
	if m.Ctx.Request.Method == "GET" {
		m.Data["type"] = requestType
		m.displayActive("country/list.html", menu)
		return
	}

	start, length := m.GetAjaxPageParams()
	log.Info(" 列表请求 start=%v length=%v search=%v sort=%v order=%v", start, length, m.getEscapeString("search"),
		m.getEscapeString("sort"), m.getEscapeString("order"))

	countryStats := m.getCountries(requestType)

	//过滤搜索关键字，只支持名称
	search := m.getEscapeString("search")
	if len(search) > 0 {
		countryStats2 := make([]*CountryStat, 0, len(countryStats))
		for _, countryStat := range countryStats {
			if strings.Contains(countryStat.Name, search) {
				countryStats2 = append(countryStats2, countryStat)
			}
		}
		countryStats = countryStats2
	}

	////临时测试
	//countryStats2 := make([]*CountryStat, 0, 60)
	//for i := 0; i < 60; i++ {
	//	for _, countryStat := range countryStats {
	//		countryStats2 = append(countryStats2, countryStat)
	//	}
	//}
	//countryStats = countryStats2

	sortStr := m.getEscapeString("sort")
	order := m.getEscapeString("order")
	var sortFunc func(c1, c2 *CountryStat) bool = nil
	switch sortStr {
	case "IpCount":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.IpCount < c2.IpCount
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.IpCount > c2.IpCount
			}
		}
	case "ConnectCount":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.ConnectCount < c2.ConnectCount
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.ConnectCount > c2.ConnectCount
			}
		}
	case "Rtt":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.Rtt < c2.Rtt
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.Rtt > c2.Rtt
			}
		}
	case "MinTTL":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.MinTTL < c2.MinTTL
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.MinTTL > c2.MinTTL
			}
		}
	case "RetransmissionRate":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.RetransmissionRate < c2.RetransmissionRate
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.RetransmissionRate > c2.RetransmissionRate
			}
		}
	case "Rps":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.Rps < c2.Rps
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.Rps > c2.Rps
			}
		}
	case "MinWindowSize":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.MinWindowSize < c2.MinWindowSize
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.MinWindowSize > c2.MinWindowSize
			}
		}

	case "ConnectTimeAverage":
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.ConnectTimeAverage < c2.ConnectTimeAverage
			}
		} else {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.ConnectTimeAverage > c2.ConnectTimeAverage
			}
		}

	default:
		if order == "asc" {
			sortFunc = func(c1, c2 *CountryStat) bool {
				return c1.UserCount < c2.UserCount
			}
		} else {
			sortFunc = byUserCountDesc
		}

	}
	sort.Slice(countryStats, func(i, j int) bool {
		return sortFunc(countryStats[i], countryStats[j])
	})

	end := start + length
	if end > len(countryStats) {
		end = len(countryStats)
	}
	returnStats := countryStats[start:end]

	//数据有限全部返回展示，不做分页，客户端自己进行排序
	m.AjaxTable(returnStats, len(returnStats), len(countryStats), nil)

}

// 获取国家列表并做统计
func (m *CountryController) getCountries(requestType int) []*CountryStat {
	collection := manager.GetDataManager().GetDB().Database(config.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//只统计一天的数据
	beginTime := util.CurrentTimeMillisecond() - 24*3600*1000
	query := bson.M{"beginTime": bson.M{"$gt": beginTime}}
	cur, err := collection.Find(ctx, query)

	if err != nil {
		log.Error("%v", err)
		return make([]*CountryStat, 0, 1)
	}
	countryStats := make(map[string]*CountryStat, 100)
	socketStats := make([]*mode.SocketStat, 0, 1000)
	for cur.Next(ctx) {
		var socketStat = &mode.SocketStat{}
		err := cur.Decode(&socketStat)
		if err != nil {
			log.Fatal("%v", err)
		}
		socketStats = append(socketStats, socketStat)
		city := manager.GetDataManager().GetCity(socketStat.Connection.Local.Ip)
		if city == nil {
			continue
		}

		if city.Country.Names == nil { //本地可能地址获取失败，用服务器代替
			city = manager.GetDataManager().GetCity(config.ApplicationConfigInstance.ServerIp)
		}
		key := city.Country.Names["zh-CN"]
		if requestType == 1 {
			key = fmt.Sprintf("%v-%v", key, city.City.Names["en"])
		}
		stat := countryStats[key]
		if stat == nil {
			stat = &CountryStat{Name: key, ips: make(map[string]bool, 10), users: make(map[string]bool, 10)}
			stat.MinTTL = math.MaxUint8
			stat.MinWindowSize = math.MaxInt16
			stat.PacketStat = &mode.PacketStat{}
			stat.MessageStat = &mode.MessageStat{}
			countryStats[key] = stat
		}
		stat.add(socketStat)
	}

	stats := make([]*CountryStat, 0, len(countryStats))
	for _, stat := range countryStats {
		stat.average()
		stats = append(stats, stat)
	}
	return stats
}

// CountryStat web展示的国家或城市 列表信息
type CountryStat struct {
	mode.SocketStat
	Name                      string          //唯一标识ID
	IpCount                   int             //ip数
	UserCount                 int             //用户数
	ConnectCount              int             //连接数
	RetransmissionRate        float32         // 重传率（总的）
	Rps                       float32         //rps 总的
	ByteSize                  string          //字节大小（B，KB、MB） 上行消息字节数/包字节数  下行消息字节数/包字节数
	ConnectTimeAverage        int             //TCP平均连接时间
	PacketCount               string          //包数 (消息数/包数/重传包数)
	Duration                  int32           //持续时间s
	ExecuteTime               int32           //总执行时间
	ExecuteAverageTime        int32           //平均执行时间 ms
	SeqRequestCount           int32           //序列号请求消息数
	MessageMaxSize            uint32          //消息最大长度
	TimeOutCount              int32           //超时数
	PacketRetransmissionRate  string          //包重传率
	MessageRetransmissionRate string          //消息重传率
	Rps2                      string          //上下行rps
	ReconnectCount            uint32          //重连数
	LoginCount                uint32          //登录数
	ips                       map[string]bool //ip详细
	users                     map[string]bool //用户详细
}

// 求平均值
func (c *CountryStat) average() {
	c.IpCount = len(c.ips)
	c.UserCount = len(c.users)
	c.RetransmissionRate = c.PacketStat.RetransmissionRate() * 100
	c.Rps = c.MessageStat.Rps(mode.DaySecond)
	c.ByteSize = fmt.Sprintf("%v/%v&nbsp;&nbsp;&nbsp;%v/%v", util.ByteConvertString(float32(c.MessageStat.UploadBytes)),
		util.ByteConvertString(float32(c.PacketStat.UploadBytes)), util.ByteConvertString(float32(c.MessageStat.DownloadBytes)),
		util.ByteConvertString(float32(c.PacketStat.DownloadBytes)))
	c.ConnectTimeAverage = int(c.Duration) / c.ConnectCount
	p := c.PacketStat
	c.PacketCount = fmt.Sprintf("%v/%v/%v&nbsp;&nbsp;&nbsp;%v/%v/%v", c.MessageStat.UploadCount, p.UploadPackets, p.UploadRetransmissionCount,
		c.MessageStat.DownloadCount, p.DownloadPackets, p.DownRetransmissionCount)
	if c.SeqRequestCount > 0 {
		c.ExecuteAverageTime = c.ExecuteTime / c.SeqRequestCount
		messageRetransmissionRate := float32(c.TimeOutCount) / float32(c.SeqRequestCount)
		c.MessageRetransmissionRate = fmt.Sprintf("%.2f%%", messageRetransmissionRate*100)
	}
	c.PacketRetransmissionRate = fmt.Sprintf("%.2f%%/%.2f%%", c.PacketStat.UploadRetransmissionRate()*100, c.PacketStat.DownloadRetransmissionRate()*100)
	c.Rps2 = fmt.Sprintf("%.2f/%.2f", c.MessageStat.UploadRps(mode.DaySecond), c.MessageStat.DownloadRps(mode.DaySecond))
	c.Rtt = c.Rtt / uint32(c.ConnectCount)
}

func (c *CountryStat) add(stat *mode.SocketStat) {
	c.ips[stat.Connection.Local.Ip] = true
	c.ConnectCount++
	c.PacketStat.Add(stat.PacketStat)
	c.MessageStat.Add(stat.MessageStat)
	c.Duration += stat.DurationSecond()
	c.Rtt += stat.Rtt
	if c.MessageMaxSize < stat.MessageStat.MaxBytes {
		c.MessageMaxSize = stat.MessageStat.MaxBytes
	}
	if c.MinTTL > stat.MinTTL {
		c.MinTTL = stat.MinTTL
	}
	if c.MinWindowSize > stat.MinWindowSize {
		c.MinWindowSize = stat.MinWindowSize
	}

	appStat := stat.MessageStat.AppStat
	if appStat != nil {
		st := appStat.(primitive.D)
		m := st.Map()
		c.TimeOutCount += m["timeOutCount"].(int32)
		c.ExecuteTime += m["seqExecuteTime"].(int32)
		c.SeqRequestCount += m["seqCount"].(int32)
		if m["reconnect"] != nil && m["reconnect"].(bool) {
			c.ReconnectCount++
		} else {
			c.LoginCount++
		}
		if m["playerId"] != nil {
			playerId := m["playerId"].(string)
			if len(playerId) > 0 {
				c.users[playerId] = true
			}
		}
	}
}

// CountryStat 用户倒序排序
func byUserCountDesc(c1, c2 *CountryStat) bool {
	return c1.UserCount > c2.UserCount
}

//======================================================================================================================
//chart 图表

// Chart 图表显示
func (m *CountryController) Chart() {
	requestType := m.GetIntNoErr("type", 0)
	menu := "country"
	if requestType == 1 {
		menu = "city"
	}
	log.Info("请求类型：%v", requestType)
	if m.Ctx.Request.Method == "GET" {
		m.Data["type"] = requestType
		m.displayActive("country/chart.html", menu)
		return
	}

	countryStats := m.getCountries(requestType)
	type Result struct {
		Countries       []string  `json:"countries"`
		UserCounts      []int     `json:"userCounts"`
		IpCounts        []int     `json:"ipCounts"`
		ConnectCounts   []int     `json:"connectCounts"`
		Rtts            []uint32  `json:"rtts"`
		Retransmissions []float32 `json:"retransmissions"`
		ConnectTimes    []int     `json:"connectTimes"`
	}
	sort.Slice(countryStats, func(i, j int) bool {
		return byUserCountDesc(countryStats[i], countryStats[j])
	})
	// 按人数排序，最多显示50条
	if len(countryStats) > 50 {
		countryStats = countryStats[:50]
	}

	length := len(countryStats)
	result := &Result{
		Countries:       make([]string, 0, length),
		UserCounts:      make([]int, 0, length),
		IpCounts:        make([]int, 0, length),
		ConnectCounts:   make([]int, 0, length),
		Rtts:            make([]uint32, 0, length),
		Retransmissions: make([]float32, 0, length),
		ConnectTimes:    make([]int, 0, length),
	}
	for _, stat := range countryStats {
		result.Countries = append(result.Countries, stat.Name)
		result.UserCounts = append(result.UserCounts, stat.UserCount)
		result.IpCounts = append(result.IpCounts, stat.IpCount)
		result.ConnectCounts = append(result.ConnectCounts, stat.ConnectCount)
		result.Rtts = append(result.Rtts, stat.Rtt)
		result.Retransmissions = append(result.Retransmissions, stat.RetransmissionRate*100)
		result.ConnectTimes = append(result.ConnectTimes, stat.ConnectTimeAverage)
	}
	m.Data["json"] = result
	m.ServeJSON()
	m.StopRun()

}

//======================================================================================================================
//Map 地图页面

// Map 地图显示
func (m *CountryController) Map() {
	requestType := m.GetIntNoErr("type", -1)
	log.Debug("请求类型：%v", requestType)

	//标记服务器位置
	serverCity := manager.GetDataManager().GetCity(config.ApplicationConfigInstance.ServerIp)
	log.Info("serverCity=%s-%s %v %v", serverCity.Country.Names["zh-CN"], serverCity.City.Names["en"],
		serverCity.Location.Longitude, serverCity.Location.Latitude)

	m.Data["serverLocationName"] = fmt.Sprintf("%v-%v", serverCity.Country.Names["zh-CN"], serverCity.City.Names["en"])
	m.Data["serverLocation"] = serverCity.Location
	m.Data["type"] = requestType

	menu := "country"
	if requestType == 1 { //城市
		menu = "city"
	}
	m.displayActive("country/map.html", menu)
}

// MapData 获取地图数据
func (m *CountryController) MapData() {
	requestType := m.GetIntNoErr("type", -1)
	log.Info("请求类型：%v", requestType)
	m.getMapData(requestType)
	m.Data["json"] = m.getMapData(requestType)

	m.ServeJSON()
	m.StopRun()
}

// 地图展示数据 requestType=0国家 1城市
func (m *CountryController) getMapData(requestType int) *MapData {
	data := &MapData{}
	//查询数据
	collection := manager.GetDataManager().GetDB().Database(config.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//只统计一天的数据
	beginTime := util.CurrentTimeMillisecond() - 24*3600*1000
	query := bson.M{"beginTime": bson.M{"$gt": beginTime}}
	cur, err := collection.Find(ctx, query)
	socketStats := make([]*mode.SocketStat, 0, 1000)
	if err != nil {
		log.Error("%v", err)
		return data
	}
	groups := make(map[string]*MapGroup)
	for cur.Next(ctx) {
		var socketStat = &mode.SocketStat{}
		err := cur.Decode(&socketStat)
		if err != nil {
			log.Fatal("%v", err)
		}
		socketStats = append(socketStats, socketStat)
		city := manager.GetDataManager().GetCity(socketStat.Connection.Local.Ip)
		if city == nil {
			continue
		}

		if city.Country.Names == nil { //本地可能地址获取失败，用服务器代替
			city = manager.GetDataManager().GetCity(config.ApplicationConfigInstance.ServerIp)
		}
		key := city.Country.Names["zh-CN"]
		if requestType == 1 {
			key = fmt.Sprintf("%v-%v", key, city.City.Names["en"])
		}
		group := groups[key]
		if group == nil {
			group = &MapGroup{Name: key, MinTTL: math.MaxUint8}
			groups[key] = group
		}
		group.add(socketStat, city)
	}
	groups2 := make([]*MapGroup, 0, len(groups))
	for _, group := range groups {
		group.average()
		groups2 = append(groups2, group)
	}
	data.Groups = groups2

	return data
}

// MapData 地图展示数据
type MapData struct {
	Groups []*MapGroup `json:"groups"` //分组数据
}

// MapGroup 地图分组（国家或城市）
type MapGroup struct {
	Name           string  `json:"name"`           //名称
	Latitude       float64 `json:"latitude"`       //维度
	Longitude      float64 `json:"longitude"`      //经度
	Count          uint32  `json:"count"`          //数量
	Rtt            uint32  `json:"rtt"`            //往返时间
	RetransmitRate float32 `json:"retransmitRate"` //包重传率
	MinTTL         uint8   `json:"minTTL"`         //最小RTT
}

// 累加
func (g *MapGroup) add(stat *mode.SocketStat, city *geoip2.City) {
	g.Latitude += city.Location.Latitude
	g.Longitude += city.Location.Longitude
	g.Count++
	g.Rtt += stat.Rtt
	g.RetransmitRate += stat.PacketStat.RetransmissionRate()
	if g.MinTTL > stat.MinTTL {
		g.MinTTL = stat.MinTTL
	}
}

// 求平均值
func (g *MapGroup) average() {
	g.Latitude = g.Latitude / float64(g.Count)
	g.Longitude = g.Longitude / float64(g.Count)
	g.Rtt = g.Rtt / g.Count
	g.RetransmitRate = g.RetransmitRate / float32(g.Count)
}
