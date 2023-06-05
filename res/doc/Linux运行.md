# Linux运行
&emsp;&emsp;`service/gate/example`目录提供了示例TCP服务器客户端，可直接本地运行。需要提前安装go和mongodb运行环境。


### 安装MongoDB 
```shell
# 1. 运行镜像
docker run -itd --name mongo-3.6.23 --restart=always -p 27017:27017 mongo:3.6.23 --auth

# 2. 进入容器创建账号
docker exec -it mongo-3.6.23 mongo admin
# 创建一个名为 admin，密码为 123456 的用户。
db.createUser({ user:'admin',pwd:'123456',roles:[ { role:'userAdminAnyDatabase', db: 'admin'},"readWriteAnyDatabase"]});
# 尝试使用上面创建的用户信息进行连接。
db.auth('admin', '123456')

# 3. 连接地址
mongodb://admin:123456@192.168.110.16:27017/?authSource=admin&readPreference=primary&appname=MongoDB%20Compass&ssl=false
```
### 运行项目
**项目地址：**<http://127.0.0.1:5041>   
**账号：** admin  
**密码：** 123  
```shell
# 1. 克隆项目
git clone https://github.com/jzyong/TcpMonitor.git

# 2. 构建项目
cd TcpMonitor
CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
go build

# 3. 查看网卡及配置config/ApplicationConfig_develop_gate.json 中的device
./TcpMonitor -m device
  [INFO]net_manager.go(105)--> 网卡：eth0 ==>

# 4 运行网络工具 访问：http://127.0.0.1:5041 
./TcpMonitor --config config/ApplicationConfig_develop_gate.json
```

### 运行TCP服务器、客户端
```shell
# 1.运行测试服务器代码
cd service\gate\example
go test
```


## 常见问题
**1. error while loading shared libraries: libpcap.so.1: cannot open shared object file: No such file or directory**
```shell
sudo yum install libpcap
```
