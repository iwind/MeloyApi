package MeloyApi

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"regexp"
	"time"
	"strings"
	"math/rand"
	"log"
	"fmt"
	"strconv"
	"bytes"
	"os"
	"syscall"
	"os/signal"
)

const MELOY_API_VERSION = "1.0"

type AppManager struct {
	AppDir string
	IsDebug bool
}

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

type ApiConfig struct {
	cacheTags []string
	cacheLifeMs int64
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
var adminManager AdminManager
var statManager StatManager
var cacheManager CacheManager
var hookManager HookManager
var appManager AppManager
var requestClient = &http.Client{}
var apiHandlers ApiHandlers = ApiHandlers{}
var serverMux = http.NewServeMux()

// 加载应用
func Start(appDir string) {
	appManager.AppDir = appDir

	if appManager.isCommand() {
		return
	}

	//写入PID
	err := ioutil.WriteFile(appDir + "/data/pid", bytes.NewBufferString(strconv.Itoa(os.Getpid())).Bytes(), 0644)
	if err != nil {
		log.Fatal(err)
		return
	}

	//日志
	if !appManager.IsDebug {
		logFile, err := os.OpenFile(appManager.AppDir + "/logs/meloy.log", os.O_APPEND | os.O_WRONLY | os.O_CREATE, os.ModeAppend)
		if err != nil {
			log.Fatal(err)
			return
		}

		defer logFile.Close()
		log.SetOutput(logFile)
	}

	//重载信号
	signalsChannel := make(chan os.Signal, 1024)
	signal.Notify(signalsChannel, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		for {
			sig := <-signalsChannel
			if sig == syscall.SIGHUP {
				appManager.reload()
			} else {
				os.Exit(0)
			}
		}
	}()

	//初始化统计管理器
	statManager.Init(appDir)

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

	address := fmt.Sprintf("%s:%d", app.Host, app.Port)
	log.Printf("start %s:%d\n", app.Host, app.Port)

	//启动Server

	go func () {
		appManager.reload()

		http.ListenAndServe(address, serverMux)
	}()

	//启动Admin
	adminManager.Load(appDir)

	//启动缓存
	cacheManager.Init()

	//等待请求
	appManager.Wait()
}

// 判断是否为命令
func (manager *AppManager) isCommand() (isCommand bool) {
	isCommand = true

	if len(os.Args) > 1 {
		command := os.Args[1]
		if command == "start" {
			manager.StartCommand()
			return
		} else if command == "stop" {
			manager.StopCommand()
			return
		} else if command == "restart" {
			manager.RestartCommand()
			return
		} else if command == "reload" {
			manager.ReloadCommand()
			return
		} else if command == "debug" {
			manager.IsDebug = true
			isCommand = false
			return
		} else if command == "help" {
			manager.HelpCommand()
			return
		} else if command == "version" {
			manager.VersionCommand()
			return
		}

		log.Println("unsupported args '" + strings.Join(os.Args[1:], " ") + "'")
		return
	}

	isCommand = false

	return
}

// 在后端运行
func (manager *AppManager) StartCommand() {
	//是否已经有进程
	running, pid := manager.checkProcessRunning()
	if running {
		log.Fatal("the proccess is already running, pid:", pid)
		return
	}

	var attr os.ProcAttr
	process, err := os.StartProcess(os.Args[0], []string{}, &attr)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Println("start:", process.Pid)
}

// 停止进程
func (manager *AppManager) StopCommand()  {
	log.Println("stopping the server ...")

	process, err := manager.findRunningProcess()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = process.Kill()
	if err != nil {
		log.Fatal(err)
		return
	}

	pidFile := appManager.AppDir + "/data/pid"
	os.Remove(pidFile)

	log.Println("ok")
}

// 重启服务
func (manager *AppManager) RestartCommand()  {
	manager.StopCommand()
	time.Sleep(time.Microsecond * 100)
	manager.StartCommand()
}

// 重新加载Api
func (manager *AppManager) ReloadCommand() {
	process, err := manager.findRunningProcess()
	if err != nil {
		log.Println(err)
		return
	}

	err = process.Signal(syscall.SIGHUP)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("api reloaded successfully")
}

// 打印帮助
func (manager *AppManager) HelpCommand()  {
	fmt.Println(`Usage:
  ./meloy-api
  	Start server sliently

  ./meloy-api debug
  	Start server in debug mode

  ./meloy-api start
  	Start server in system background

  ./meloy-api stop
  	Stop server

  ./meloy-api restart
  	Restart server

  ./meloy-api reload
  	Reload api configurations

  ./meloy-api version
  	Show api version

  ./meloy-api help
  	Show this help`)

}

// 打印版本信息
func (manager *AppManager) VersionCommand()  {
	fmt.Println("  MeloyAPI v" + MELOY_API_VERSION)
	fmt.Println("  GitHub: https://github.com/iwind/MeloyApi")
	fmt.Println("  Author: Liu Xiang Chao")
	fmt.Println("  QQ: 19644627")
	fmt.Println("  E-mail: 19644627@qq.com")
}

// 重新加载API配置
func (manager *AppManager) reload() {
	servers := appManager.loadServers(manager.AppDir)
	ApiArray = appManager.loadApis(manager.AppDir, servers)

	for _, handler := range apiHandlers {
		handler.Enabled = false
	}

	for _, api := range ApiArray {
		log.Println("load api '" + api.Path + "' from '" + api.File + "'")

		func (api Api) {
			handler, ok := apiHandlers[api.Path]
			if ok {
				handler.Enabled = true
				manager.copyApi(handler.Api, api)
			} else {
				apiHandlers.HandleFunc(serverMux, &api, func(writer http.ResponseWriter, request *http.Request) {
					appManager.handle(writer, request, &api)
				})
			}
		}(api)
	}
}

// 等待处理请求
func (manager *AppManager) Wait()  {
	defer statManager.Close()

	//Hold住进程
	for {
		time.Sleep(1 * time.Hour)
	}
}

// 加载服务器列表
func (manager *AppManager) loadServers(appDir string) (servers []Server) {
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
func (manager *AppManager) loadApis(appDir string, servers []Server) (apis []Api) {
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

		_bytes, err := ioutil.ReadFile(appDir + "/apis/" + file.Name())
		if err != nil {
			log.Printf("Error:%s:%s\n", file.Name(), err)
			continue
		}

		var api Api
		jsonError := json.Unmarshal(_bytes, &api)
		if jsonError != nil {
			log.Printf("Error:%s:%s\n", file.Name(), jsonError)
			continue
		}
		api.File = file.Name()

		//校验和转换api.methods
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
			_bytes, err := ioutil.ReadFile(dataFileName)
			if err != nil {
				log.Println("Error:" + err.Error())
			} else {
				api.Mock = string(_bytes)
			}
		}

		apis = append(apis, api)
	}

	return
}

// 处理请求
func (manager *AppManager) handle(writer http.ResponseWriter, request *http.Request, api *Api) {
	countAddresses := len(api.Addresses)

	if countAddresses == 0 {
		fmt.Fprintln(writer, "Does not have avaiable address")
		return
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Int() % countAddresses

	//检查method
	method := strings.ToUpper(request.Method)
	if !containsString(api.Methods, method) {
		fmt.Fprintln(writer, "'" + request.Method + "' method is not supported")
		return
	}

	address := api.Addresses[index]

	hookManager.beforeHook(writer, request, api, func () {
		//开始处理
		manager.handleMethod(writer, request, api, address, method)
	})
}

// 转发某个方法的请求
func (manager *AppManager) handleMethod(writer http.ResponseWriter, request *http.Request, api *Api, address ApiAddress, method string) {
	t := time.Now().UnixNano()

	query := request.URL.RawQuery

	//是否有缓存
	cacheKey := request.URL.RequestURI()
	cacheEntry, ok := cacheManager.Get(cacheKey)
	if ok {
		for key, values := range cacheEntry.Header {
			for _, value := range values {
				writer.Header().Add(key, value)
			}
		}

		writer.Write(cacheEntry.Bytes)

		statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, 0, 1)

		return
	}


	url := address.URL
	if len(query) > 0 {
		url += "?" + query
	}

	newRequest, err := http.NewRequest(method, url, nil)

	if err != nil {
		hookManager.afterHook(writer, request, nil, api, err)
		statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, 1, 0)
		return
	}

	newRequest.Header = request.Header
	newRequest.Header.Set("Meloy-Api", "1.0")
	newRequest.Body = request.Body
	resp, err := requestClient.Do(newRequest)

	if err != nil {
		log.Println("Error:" + err.Error())
		hookManager.afterHook(writer, request, nil, api, err)

		//统计
		statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, 1, 0)
		return
	}

	//调用钩子
	hookManager.afterHook(writer, request, resp, api, nil)

	//分析头部指令等信息
	apiConfig := ApiConfig{
		cacheTags: []string{ "$MeloyAPI$" + api.Path },
	}
	manager.parseResponseHeaders(writer, request, resp, address, api.Path, &apiConfig)

	_bytes, err := ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	if err != nil {
		log.Println("Error:" + err.Error())
		statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, 1, 0)
		return
	}

	//缓存
	if apiConfig.cacheLifeMs > 0 {
		cacheManager.Set(cacheKey, apiConfig.cacheTags, _bytes, writer.Header(), apiConfig.cacheLifeMs)
	}

	writer.Write(_bytes)

	var errors int64 = 0
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		errors ++
	}


	statManager.Send(address, api.Path, (time.Now().UnixNano() - t) / 1000000, errors, 0)
}

// 分析响应头部
func (manager *AppManager) parseResponseHeaders(writer http.ResponseWriter, request *http.Request, resp *http.Response, address ApiAddress, path string, apiConfig *ApiConfig)  {
	directiveReg, _ := regexp.Compile("^Meloy-Api-(.+)")

	for key, values := range resp.Header {
		if containsString([]string { "Connection", "Server" }, key) {
			continue
		}

		//处理指令
		if directiveReg.MatchString(key) {
			directive := directiveReg.FindStringSubmatch(key)[1]
			manager.processDirective(request, address, path, directive, values[0], apiConfig)

			continue
		}

		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}

	writer.Header().Set("Server", "MeloyApi")

}

// 处理指令
func (manager *AppManager) processDirective(request *http.Request, address ApiAddress, path string, directive string, value string, apiConfig *ApiConfig) {
	//调试信息
	{
		reg, _ := regexp.Compile("^Debug")
		if reg.MatchString(directive) {
			statManager.SendDebug(address, path, request.URL.RequestURI(), value)
			return
		}
	}

	//设置缓存时间
	{
		reg, _ := regexp.Compile("^Cache-Life-Ms")
		if reg.MatchString(directive) {
			life, err := strconv.Atoi(value)
			if err != nil {
				log.Println("Cache life directive Error:" + err.Error())
				return
			}
			apiConfig.cacheLifeMs = int64(life)
			return
		}
	}

	//设置缓存标签
	{
		reg, _ := regexp.Compile("^Cache-Tag")
		if reg.MatchString(directive) {
			if apiConfig.cacheTags == nil {
				apiConfig.cacheTags = []string{}
			}
			apiConfig.cacheTags = append(apiConfig.cacheTags, value)

			return
		}
	}

	//设置要删除的缓存标签
	{
		reg, _ := regexp.Compile("^Cache-Delete")
		if reg.MatchString(directive) {
			cacheManager.DeleteTag(value)

			return
		}
	}

	log.Println("Unknown directive:" + directive + " value:" + value)
}

// 检查进程是否存在
func (manager *AppManager) checkProcessRunning() (running bool, pid int) {
	pidFile := appManager.AppDir + "/data/pid"
	_bytes, err := ioutil.ReadFile(pidFile)
	if err != nil {
		running = false
		return
	}

	if len(_bytes) == 0 {
		running = false
		return
	}

	pidString := string(_bytes)
	pid, err = strconv.Atoi(pidString)
	if err != nil {
		running = false
		return
	}
	_, err = os.FindProcess(pid)
	if err != nil {
		running = false
		return
	}
	running = true
	return
}

// 拷贝API信息
func (manager *AppManager) copyApi(to *Api, from Api) {
	to.Path = from.Path
	to.Methods = from.Methods
	to.Address = from.Address

	to.Name = from.Name
	to.Description = from.Description
	to.Params = from.Params
	to.Dones = from.Dones
	to.Todos = from.Todos
	to.IsDeprecated = from.IsDeprecated
	to.Version = from.Version

	to.Addresses = from.Addresses
	to.File = from.File
	to.Mock = from.Mock

	to.Stat = from.Stat
}

func (manager *AppManager) findRunningProcess() (process *os.Process, err error) {
	pidFile := appManager.AppDir + "/data/pid"
	_bytes, err := ioutil.ReadFile(pidFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	if len(_bytes) == 0 {
		log.Println("ok")
		return
	}

	pidString := string(_bytes)
	pid, err := strconv.Atoi(pidString)
	log.Println("pid:", pid)
	if err != nil {
		log.Fatal(err)
		return
	}
	process, err = os.FindProcess(pid)
	if err != nil {
		log.Fatal(err)
		return
	}

	return
}