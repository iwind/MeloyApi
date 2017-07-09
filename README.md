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

	//启用应用
	MeloyApi.LoadApp(appDir)
}
~~~