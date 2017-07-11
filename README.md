# Meloy API
使用GO语言实现的API网关。

# 使用
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

# 编译好的Server
可以在 https://github.com/iwind/MeloyApiServer 找到已经编译好的二进制文件和项目目录结构，可以直接clone使用。