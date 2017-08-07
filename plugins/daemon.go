package plugins

import (
	"io/ioutil"
	"log"
	"regexp"
	"encoding/json"
	"os/exec"
)

// 监控文件
type DaemonPlugin struct {
}

// 配置
type DaemonConfig struct {
	Commands []string `json:"commands"`
}

func (plugin *DaemonPlugin) Name() string {
	return "daemon"
}

func (plugin *DaemonPlugin) StartFunc(appDir string) {
	var bytes, err = ioutil.ReadFile(appDir + "/config/plugin_daemon.json")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	commentReg, err := regexp.Compile("/([*]+((.|\n|\r)+?)[*]+/)|(\n\\s+//.+)")
	var jsonString = string(bytes)
	if err != nil {
		log.Printf("Error:%s", err)
	} else {
		jsonString = commentReg.ReplaceAllString(jsonString, "")
	}

	var config = DaemonConfig{}
	err = json.Unmarshal([]byte(jsonString), &config)
	if err != nil {
		log.Printf("Error:%s:\n~~~\n%s\n~~~\n", err, jsonString)
		return
	}

	log.Println(config.Commands)
	if len(config.Commands) == 0 {
		return
	}

	for _, command := range config.Commands {
		go func(command string) {
			for {
				log.Println("Deamon Log:start command:", command)
				var cmd = exec.Command("sh", "-c", command)

				if err != nil {
					log.Println("Deamon Log:", err.Error())
					return
				}

				err = cmd.Start()

				if err != nil {
					log.Println("Deamon Log:", err.Error())
					cmd.Wait()
					return
				}

				cmd.Wait()
			}
		}(command)
	}
}

func (plugin *DaemonPlugin) ReloadFunc() {

}

func (plugin *DaemonPlugin) StopFunc() {

}
