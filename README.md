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
	pwd, _  := os.Getwd()

	configDir := pwd + "/apps/meloy"
	webDir := pwd + "/web"

	//启用应用
	MeloyApi.LoadApp(configDir)

	//启用UI
	MeloyApi.LoadUI(webDir)

	//等待进行
	MeloyApi.Wait()
}
~~~