# Meloy API
使用GO语言实现的API网关。

# 编译好的Server
可以在 https://github.com/iwind/MeloyApiServer 找到已经编译好的二进制文件和项目目录结构，可以直接clone使用。

# 创建自己的网关
~~~go
package main

import (
	"os"
	"github.com/iwind/MeloyApi"
)

func main()  {
	//加载服务器
	appDir, _  := os.Getwd()

	//启动应用
	MeloyApi.Start(appDir)
}
~~~

# 性能
* CPU x 2/4G内存：2800 requests/s
* CPU x 8/8G内存：5500 requests/s

使用的测试脚本：
~~~shell
ab -c 1000 -n 10000 "http://192.168.1.233:8000/s"
~~~


# 文档
[点击这里查看文档](./docs/SUMMARY.md)