package controller

import (
	"github.com/beego/beego/v2/server/web"
	config2 "github.com/jzyong/TcpMonitor/config"
)

// User 用户数据，存session
type User struct {
}

type UserController struct {
	web.Controller
}

func (u *UserController) Index() {
	u.TplName = "user/index.html"
	u.Data["web_base_url"] = "" //必须设置，不然web 获取url==null
	u.Render()
}

// Login curl --data "name=jzy&password=123" http://127.0.0.1:5041/user/login
func (u *UserController) Login() {
	username := u.GetString("username")
	password := u.GetString("password")

	config := config2.ApplicationConfigInstance
	if config.WebAccount == username && config.WebPassword == password {
		u.Data["json"] = map[string]interface{}{"status": 1, "msg": "login success"}
		u.SetSession("isAdmin", true)
		u.SetSession("auth", true)
		u.SetSession("user", &User{})

	} else {
		u.Data["json"] = map[string]interface{}{"status": 0, "msg": "username or password incorrect"}
		u.SetSession("isAdmin", false)
		u.SetSession("auth", false)
	}
	u.ServeJSON()
}

// Logout 退出
func (u *UserController) Logout() {
	u.SetSession("auth", false)
	u.SetSession("isAdmin", false)
	u.DelSession("user")
	u.Redirect("/user/index", 302)
}
