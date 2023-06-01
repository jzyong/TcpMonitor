package router

import (
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	config2 "github.com/jzyong/TcpMonitor/config"
	"github.com/jzyong/TcpMonitor/web/controller"
	"github.com/jzyong/golib/log"
	"time"
)

// StartWeb 启动web
func StartWeb() {
	//设置配置
	web.BConfig.CopyRequestBody = true
	web.BConfig.WebConfig.Session.SessionOn = true
	web.BConfig.AppName = "net-service"
	web.BConfig.ServerName = "net-service"
	web.BConfig.Log.FileLineNum = true
	if config2.ApplicationConfigInstance.Profile == "jzy" {
		web.BConfig.RunMode = web.DEV //开发模式消耗更多，如每次render都构建模板等
	} else {
		web.BConfig.RunMode = web.PROD
	}
	if config2.ApplicationConfigInstance.Profile == "online" {
		logs.SetLogger(logs.AdapterFile, `{"filename":"../log/net-service-web.log","maxsize":102400000,"maxbackup":7}`)
		loc, err := time.LoadLocation("America/Atikokan")
		if err != nil {
			log.Warn("修改时区错误：%v")
		}
		time.Local = loc
	}

	//路由注册
	userController := &controller.UserController{}
	indexController := &controller.IndexController{}
	web.AutoRouter(userController)
	web.AutoRouter(indexController)
	web.AutoRouter(&controller.SocketController{})
	web.AutoRouter(&controller.ServerController{})
	web.AutoRouter(&controller.ConsoleController{})
	web.AutoRouter(&controller.CountryController{})
	web.AutoRouter(&controller.MessageController{})
	web.Router("/", indexController, "*:Index")
	//web.ErrorController(&controller.ErrorController{})

	//http://localhost:5041
	//外测服不能绑定[ip:port]形式，只能[:ip] beego的bug？还是系统配置问题？
	web.Run(config2.ApplicationConfigInstance.WebUrl)

}
