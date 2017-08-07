package plugins

import (
	"io/ioutil"
	"log"
	"encoding/json"
	"time"
	"os"
	"os/exec"
	"net/http"
	"strings"
	"regexp"
)

// 监控文件
type WatchPlugin struct {
}

// 配置
type WatchConfig struct {
	Files []WatchFileConfig `json:"files"`
}

type WatchFileConfig struct {
	File string `json:"file"`
	Notify struct {
		Command string
		URL string

		Commands []string
		URLs []string
	} `json:"notify"`

	exists     bool
	modifiedAt int64
}

func (plugin *WatchPlugin) Name() string {
	return "watch"
}

func (plugin *WatchPlugin) StartFunc(appDir string) {
	var bytes, err = ioutil.ReadFile(appDir + "/config/plugin_watch.json")
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

	var config = WatchConfig{}
	err = json.Unmarshal([]byte(jsonString), &config)
	if err != nil {
		log.Printf("Error:%s:\n~~~\n%s\n~~~\n", err, jsonString)
		return
	}

	if len(config.Files) == 0 {
		return
	}

	go func() {
		var tick = time.Tick(1 * time.Second)
		for {
			<-tick

			for index, file := range config.Files {
				stat, err := os.Stat(file.File)
				if err != nil {
					config.Files[index].exists = false
					continue
				}
				var modifiedAt = stat.ModTime().Unix()
				if file.modifiedAt == 0 {
					config.Files[index].modifiedAt = modifiedAt
					continue
				}
				if file.modifiedAt != modifiedAt {
					config.Files[index].exists = true
					config.Files[index].modifiedAt = modifiedAt
					notifyFile(file)
				}
			}
		}
	}()
}

func (plugin *WatchPlugin) ReloadFunc() {

}

func (plugin *WatchPlugin) StopFunc() {

}

// 发送通知
func notifyFile(file WatchFileConfig) {
	// 执行命令
	var cmds = file.Notify.Commands
	if len(file.Notify.Command) > 0 {
		cmds = append(cmds, file.Notify.Command)
	}

	if len(cmds) > 0 {
		for _, command := range cmds {
			func(command string) {
				go func() {
					log.Println("Watch Log: Start", command)
					var cmd = exec.Command("sh", "-c", command)

					stdout, err := cmd.StdoutPipe()
					if err != nil {
						log.Println("Watch Log:", err.Error())
						return
					}

					err = cmd.Start()

					if err != nil {
						log.Println("Watch Log:", err.Error())
						stdout.Close()
						cmd.Wait()
						return
					}

					_bytes, err := ioutil.ReadAll(stdout)
					if err != nil {
						log.Println("Watch Log:", err.Error())
						stdout.Close()
						cmd.Wait()
						return
					}

					stdout.Close()

					cmd.Wait()
					log.Println("Watch Log:", strings.Trim(string(_bytes), " \t\n"))
				}()
			}(command)
		}
	}

	//调用URL
	var urls = file.Notify.URLs
	if len(file.Notify.URL) > 0 {
		urls = append(urls, file.Notify.URL)
	}

	if len(urls) > 0 {
		for _, url := range urls {
			func(url string) {
				go func() {
					http.Get(url)
					log.Println("Watch Log: Notify", url)
				}()
			}(url)
		}
	}
}
