package manager

import (
	"context"
	"fmt"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/mode"
	"github.com/jzyong/golib/log"
	"github.com/jzyong/golib/util"
	"github.com/oschwald/geoip2-golang"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DataManager Data
type DataManager struct {
	util.DefaultModule
	client           *mongo.Client                              //生产数据库
	CreateMessageFun func(Id string, stat *mode.SocketStat) any //创建自定义消息对象
	snowFlake        *util.Snowflake                            //唯一Id生成
	ipCityDB         *geoip2.Reader                             //城市
}

var dataManager *DataManager
var dataManagerOnce sync.Once

func GetDataManager() *DataManager {
	dataManagerOnce.Do(func() {
		dataManager = &DataManager{snowFlake: &util.Snowflake{}}
	})
	return dataManager
}

// Init 开始启动
func (m *DataManager) Init() error {
	log.Info("data init started ......")
	m.snowFlake.Init(int16(config2.ApplicationConfigInstance.Id))
	go m.StartDB()
	go m.startGeoLite()

	return nil
}

// StartDB 启动生产数据库
func (m *DataManager) StartDB() error {
	// 启动机器人数据库
	client, err := mongo.NewClient(options.Client().ApplyURI(config2.ApplicationConfigInstance.DatabaseUrl))
	if err != nil {
		log.Error("%v", err)
		return err
	}
	m.client = client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = m.client.Connect(ctx)
	if err != nil {
		log.Error("%v", err)
		return err
	}
	//检测连接状态
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Error("%v", err)
		return err
	}
	go m.clearExpireData()
	return nil
}

// 启动ip数据库
func (m *DataManager) startGeoLite() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Info("项目路径：%v", cwd)
	path := filepath.Join(cwd, "config", "GeoLite2-City.mmdb")
	m.ipCityDB, err = geoip2.Open(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	//log.Info("国家城市：%v", m.GetCountyCity("182.148.56.155"))
}

// GetCountyCity 获取ip所属的国家城市
func (m *DataManager) GetCountyCity(ipStr string) string {
	ip := net.ParseIP(ipStr)
	city, err := m.ipCityDB.City(ip)
	if err != nil {
		log.Warn("%v获取城市错误：%v", ip, err)
		return ""
	}
	//城市名字选择英文，中文国外可能获取不到
	return fmt.Sprintf("%v-%v", city.Country.Names["zh-CN"], city.City.Names["en"])
}

// GetIsoCodeLAndCoordinate 获取 国家iso code 和经纬度
func (m *DataManager) GetIsoCodeLAndCoordinate(ipStr string) (string, float64, float64) {
	ip := net.ParseIP(ipStr)
	city, err := m.ipCityDB.City(ip)
	if err != nil {
		log.Warn("%v获取城市错误：%v", ip, err)
		return "", 0, 0
	}
	return city.Country.IsoCode, city.Location.Longitude, city.Location.Latitude
}

// GetCity 获取城市
func (m *DataManager) GetCity(ipStr string) *geoip2.City {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Warn("ip地址:%v 获取失败", ipStr)
		return nil
	}
	city, err := m.ipCityDB.City(ip)
	if err != nil {
		log.Warn("%v获取城市错误：%v", ip, err)
		return nil
	}
	return city
}

func (m *DataManager) GetDB() *mongo.Client {
	if m.client == nil {
		log.Warn("生产数据库 mongoDB未创建")
		return nil
	}
	return m.client
}

// StructToM 将结构体转换为更新的M
func (m *DataManager) StructToM(o interface{}) *bson.M {
	bytes, err := bson.Marshal(o)
	if err != nil {
		log.Error("%v", err)
		return nil
	}
	var update bson.M
	err = bson.Unmarshal(bytes, &update)
	if err != nil {
		log.Error("%v", err)
		return nil
	}
	delete(update, "_id") //删除主键
	return &update
}

// Stop 关闭连接
func (m *DataManager) Stop() {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := m.client.Disconnect(ctx); err != nil {
			log.Error("%v", err)
		}
	}
	if m.ipCityDB != nil {
		m.ipCityDB.Close()
	}
}

// InsertSocketStat 保存socket信息
func (m *DataManager) InsertSocketStat(stat *mode.SocketStat) {
	collection := m.GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := m.snowFlake.GetId()
	if err != nil {
		log.Error("生成id错误:%v", err)
		return
	}
	stat.Id = fmt.Sprintf("%v", id)
	result, err := collection.InsertOne(ctx, stat)
	if err != nil {
		log.Error("%v", err)
		return
	}
	if m.CreateMessageFun != nil && len(*stat.GetMessages()) > 0 && result.InsertedID != nil {
		message := m.CreateMessageFun(result.InsertedID.(string), stat)
		m.insertMessage(message)
	}
}

// FindSocketStats 查询所有
func (m *DataManager) FindSocketStats() []*mode.SocketStat {
	collection := m.GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cur, err := collection.Find(ctx, bson.D{})
	socketStats := make([]*mode.SocketStat, 0, 1000)
	if err != nil {
		log.Error("%v", err)
		return socketStats
	}

	for cur.Next(ctx) {
		var result = &mode.SocketStat{}
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal("%v", err)
		}
		socketStats = append(socketStats, result)
	}
	return socketStats
}

// 插入消息
func (m *DataManager) insertMessage(message any) {
	collection := m.GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("message")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, message)
	if err != nil {
		log.Error("%v", err)
		return
	}
}

// 清除过期时间，数据只保留30天 ,会阻塞goroutine，每间隔24小时执行一次
func (m *DataManager) clearExpireData() {
	//socket_stat
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clearTime := util.ZeroUnixTime(-mode.ExpireDay)
	filter := bson.D{
		{"beginTime", bson.D{{"$lt", clearTime}}},
	}
	collection := m.GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("socket_stat")
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		log.Error("%v", err)
	}
	if result.DeletedCount > 0 {
		log.Info("删除 socket_stat 数量:%v", result.DeletedCount)
	}

	//socket_stat
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	// message 数据量大，12000条数据就消耗1.2G存储空间，并且该消息重要性也相对低
	clearTime2 := util.ZeroUnixTime(-2)
	filter = bson.D{
		{"createTime", bson.D{{"$lt", clearTime2}}},
	}
	collection = m.GetDB().Database(config2.ApplicationConfigInstance.DatabaseName).Collection("message")
	result, err = collection.DeleteMany(ctx2, filter)
	if err != nil {
		log.Error("%v", err)
	}
	if result.DeletedCount > 0 {
		log.Info("删除 message 数量:%v", result.DeletedCount)
	}
	time.AfterFunc(time.Hour*24, func() {
		m.clearExpireData()
	})
}
