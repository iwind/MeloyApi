package MeloyApi

import (
	"io"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"regexp"
	"time"
	"strings"
	"math/rand"
	"log"
	"fmt"
)

type Host struct {
	Address string
	Weight int
}

type Server struct {
	Code string
	Hosts []Host
}

type App struct {
	Host string
	Port int
}

type Api struct {
	Path string `json:"path"`
	Address string `json:"address"`
	Methods []string `json:"methods"`

	Name string `json:"name"`
	Description string `json:"description"`
	Params []ApiParam `json:"params"`
	Dones []string `json:"dones"`
	Todos []string `json:"todos"`
	IsDeprecated bool `json:"isDeprecated"`
	Version string `json:"version"`

	Addresses []ApiAddress `json:"availableAddresses"`
	File string `json:"file"`
	Mock string `json:"mock"`

	Stat ApiStat `json:"stat"`
}

type ApiAddress struct {
	Server string
	Host string
	URL string
}

type ApiParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Description string `json:"description"`
}

var ApiArray []Api
var statManager StatManager
var requestClient = &http.Client{}

// 加载应用
func LoadApp(appDir string) {
	//初始化统计管理器
	statManager.Init(appDir)

	servers := loadServers(appDir)

	appBytes, appErr := ioutil.ReadFile(appDir + "/config/app.json")
	if appErr != nil {
		log.Printf("Error:%s\n", appErr)
		return
	}

	var app App
	jsonError := json.Unmarshal(appBytes, &app)
	if jsonError != nil {
		log.Printf("Error:%s", jsonError)
		return
	}

	ApiArray = loadApis(appDir, servers)

	address := fmt.Sprintf("%s:%d", app.Host, app.Port)
	log.Printf("start %s:%d\n", app.Host, app.Port)

	//启动Server
	go (func (apis []Api) {
		serverMux := http.NewServeMux()

		for _, api := range apis {
			log.Println("load api '" + api.Path + "' from '" + api.File + "'")

			func (api Api) {
				serverMux.HandleFunc(api.Path, func (writer http.ResponseWriter, request *http.Request) {
					handle(writer, request, api)
				})
			}(api)
		}

		http.ListenAndServe(address, serverMux)
	})(ApiArray)

	//启动Admin
	LoadAdmin(appDir)

	//等待请求
	Wait()
}

// 等待处理请求
func Wait()  {
	defer statManager.Close()

	//Hold住进程
	for {
		time.Sleep(1 * time.Hour)
	}
}

// 加载服务器列表
func loadServers(appDir string) (servers []Server) {
	serverBytes, serverErr := ioutil.ReadFile(appDir + "/config/servers.json")

	if serverErr != nil {
		log.Printf("Error:%s\n", serverErr)
		return
	}

	jsonErr := json.Unmarshal(serverBytes, &servers)
	if jsonErr != nil {
		log.Printf("Error:%s\n", jsonErr)
		return
	}
	return
}

// 加载Api列表
func loadApis(appDir string, servers []Server) (apis []Api) {
	files, err := ioutil.ReadDir(appDir + "/apis")
	if err != nil {
		log.Printf("Error:%s\n", err)
		return
	}

	reg, err := regexp.Compile("\\.json$")
	if err != nil {
		log.Printf("Error:%s\n", err)
		return
	}

	dataReg, err := regexp.Compile("\\.mock\\.json")
	if err != nil {
		log.Printf("Error:%s\n", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		//跳过假数据
		if dataReg.MatchString(file.Name()) {
			continue
		}

		if !reg.MatchString(file.Name()) {
			continue
		}

		bytes, err := ioutil.ReadFile(appDir + "/apis/" + file.Name())
		if err != nil {
			log.Printf("Error:%s:%s\n", file.Name(), err)
			continue
		}

		var api Api
		jsonError := json.Unmarshal(bytes, &api)
		if jsonError != nil {
			log.Printf("Error:%s:%s\n", file.Name(), jsonError)
			continue
		}
		api.File = file.Name()

		//@TODO 校验和转换api.methods
		for methodIndex, method := range api.Methods {
			api.Methods[methodIndex] = strings.ToUpper(method)
		}

		//转换地址
		for _, server := range servers {
			if len(server.Hosts) == 0 {
				continue
			}

			var totalWeight int = 0
			for _, host := range server.Hosts {
				if host.Weight < 0 {
					host.Weight = 0
				}
				totalWeight += host.Weight
			}

			reg, _ := regexp.Compile("%{server." + server.Code + "}")
			if !reg.MatchString(api.Address) {
				continue
			}

			for _, host := range server.Hosts {
				address := reg.ReplaceAllString(api.Address, host.Address)

				if totalWeight == 0 {
					api.Addresses = append(api.Addresses, ApiAddress {
						Server:server.Code,
						Host: host.Address,
						URL: address,
					})
					continue
				}

				weight := int(host.Weight * 10 / totalWeight)
				for i := 0; i < weight; i ++ {
					api.Addresses = append(api.Addresses, ApiAddress {
						Server:server.Code,
						Host: host.Address,
						URL: address,
					})
				}
			}
		}

		//假数据
		fileName := file.Name()
		reg, _ := regexp.Compile("\\.json")
		dataFileName := appDir + "/apis/" + reg.ReplaceAllString(fileName, ".mock.json")

		fileExists, _ := FileExists(dataFileName)
		if fileExists {
			bytes, err := ioutil.ReadFile(dataFileName)
			if err != nil {
				log.Println("Error:" + err.Error())
			} else {
				api.Mock = string(bytes)
			}
		}

		apis = append(apis, api)
	}

	return
}

// 处理请求
func handle(writer http.ResponseWriter, request *http.Request, api Api) {
	countAddresses := len(api.Addresses)

	if countAddresses == 0 {
		fmt.Fprintln(writer, "Does not have avaiable address")
		return
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Int() % countAddresses

	//检查method
	method := strings.ToUpper(request.Method)
	if !contains(api.Methods, method) {
		fmt.Fprintln(writer, "'" + request.Method + "' method is not supported")
		return
	}

	address := api.Addresses[index]

	beforeHook(writer, request, api, func () {
		//开始处理
		handleMethod(writer, request, api, address, method)
	})
}

// 转发某个方法的请求
func handleMethod(writer http.ResponseWriter, request *http.Request, api Api, address ApiAddress, method string) {
	t := time.Now().UnixNano()

	query := request.URL.RawQuery
	url := address.URL
	if len(query) > 0 {
		url += "?" + query
	}
	newRequest, err := http.NewRequest(method, url, nil)

	if err != nil {
		afterHook(writer, request, nil, api, err)
		statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, 1)
		return
	}

	newRequest.Header = request.Header
	newRequest.Header.Set("Meloy-Api", "1.0")
	newRequest.Body = request.Body
	resp, err := requestClient.Do(newRequest)

	if err != nil {
		log.Println("Error:" + err.Error())
		afterHook(writer, request, nil, api, err)

		//统计
		statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, 1)
		return
	}

	//调用钩子
	afterHook(writer, request, resp, api, nil)

	parseResponseHeaders(writer, resp, address, api.Path)

	io.Copy(writer, resp.Body)
	resp.Body.Close()

	var errors int64 = 0
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		errors ++
	}
	statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, errors)
}

// 分析响应头部
func parseResponseHeaders(writer http.ResponseWriter, resp *http.Response, address ApiAddress, path string)  {
	directiveReg, _ := regexp.Compile("^Meloy-Api-(.+)")

	for key, values := range resp.Header {
		if contains([]string { "Connection", "Server" }, key) {
			continue
		}

		//处理指令
		if directiveReg.MatchString(key) {
			directive := directiveReg.FindStringSubmatch(key)[1]
			processDirective(address, path, directive, values[0])

			continue
		}

		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}

	writer.Header().Set("Server", "MeloyApi")

}

// 处理指令
func processDirective(address ApiAddress, path string, directive string, value string) {
	//调试信息
	{
		reg, _ := regexp.Compile("^Debug")
		if reg.MatchString(directive) {
			statManager.SendDebug(address, path, value)
			return
		}
	}

	log.Println("directive:" + directive + " value:" + value)
}

// 判断slice中是否包含某个字符串
func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

