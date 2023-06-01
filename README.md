# Tcp Monitor

&emsp;&emsp;Application-oriented customized TCP message interception, monitoring, and statistics.
Because the TCP protocol of each project is different, the project cannot be directly used. 
You need to modify the TCP packet logic and application statistics by yourself.


| Directory | Description                                       |
|-----------|---------------------------------------------------|
| config    | Config file                                       |
| manager   | Network core logic                                |
| mode      | Core logical entity                               |
| service   | Customize the specific service logic(need modify) |
| static    | web js,css,image(need modify)                     |
| view      | Web page(need modify)                             |
| web       | Web logic(need modify)                            |


## Features
* Monitors all TCP messages sent and received on a specified port on a network adapter
* Check why each Socket is closed
* Calculate the number, size, and so on
* Check the delay and packet loss rate of an IP address corresponding to a country or city
* Statistics of each user's network connection, message details



## Technology and reference
* [go-packet](https://github.com/google/gopacket)  
* [beego](https://github.com/beego/beego) 
* [beego-example](https://github.com/beego/beego-example)  
* [bootstrap](https://www.bootcss.com/) 
* [echarts](https://echarts.apache.org/zh/index.html) 
* [leaflet](https://leafletjs.com/index.html) 
* [nps](https://github.com/ehang-io/nps) web template reference

### TODO
1. 文档编写
2. 测试服务器和客户端(protobuf)







