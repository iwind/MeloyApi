package MeloyApi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const MELOY_API_VERSION = "1.0"

type AppManager struct {
	AppDir  string
	IsDebug bool
}

type Host struct {
	Address string
	Weight  int
}

type Server struct {
	Code  string
	Hosts []Host

	Request struct {
		Timeout string
		MaxSize string

		timeoutDuration time.Duration
		maxSizeBits     float64
	}
}

// APP配置
type AppConfig struct {
	Host string
	Port int
	SSL struct {
		Cert string
		Key  string
	}

	Allow struct {
		Clients []string
	}

	Deny struct {
		Clients []string
	}

	Limits struct {
		Requests struct {
			Minute int
			Day    int
		}
	}

	Users []struct {
		Type     string
		Username string
		Password string
	}

	// 插件
	Plugins []string

	// 是否有限制
	hasAllow bool
	hasDeny  bool

	// 监听
	isWatching bool
	watchingAt int64

	// 限流
	hasMinuteLimit bool
	hasDayLimit    bool

	limitLastMinute string
	limitLastDay    string

	limitMinuteLeft int
	limitDayLeft    int

	// 用户限制
	hasUsers bool
}

// API配置
type ApiConfig struct {
	cacheTags   []string
	cacheLifeMs int64
}

// API地址
type ApiAddress struct {
	Server string `json:"server"`
	Host   string `json:"host"`
	URL    string `json:"url"`
}

var ApiArray []Api
var appConfig AppConfig
var adminManager AdminManager
var statManager StatManager
var cacheManager CacheManager
var pluginManager PluginManager
var hookManager HookManager
var appManager AppManager
var requestClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 256,
	},
}
var handlerManager HandlerManager = HandlerManager{}
var serverMux = http.NewServeMux()
var serverMuxLoaded = false

// 加载应用
func Start(appDir string) {
	appManager.AppDir = appDir

	// 检查data/, logs/目录是否存在
	for _, dir := range []string{"data", "logs"} {
		systemDir := appDir + string(os.PathSeparator) + dir
		if exists, _ := FileExists(systemDir); !exists {
			log.Println("create dir '" + dir + "'")
			os.Mkdir(systemDir, 0777)
		}
	}

	if appManager.isCommand() {
		return
	}

	// 写入PID
	err := ioutil.WriteFile(appDir+"/data/pid", []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		log.Fatal(err)
		return
	}

	// 日志
	if len(os.Args) != 1 && !appManager.IsDebug {
		logFile, err := os.OpenFile(appManager.AppDir+"/logs/meloy.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend)
		if err != nil {
			log.Fatal(err)
			return
		}

		defer logFile.Close()
		log.SetOutput(logFile)
	}

	// 重载信号
	signalsChannel := make(chan os.Signal, 1024)
	signal.Notify(signalsChannel, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		for {
			sig := <-signalsChannel
			if sig == syscall.SIGHUP {
				appManager.reload()
			} else if sig == syscall.SIGTERM {
				pluginManager.Stop()
			} else {
				pidFile := appManager.AppDir + "/data/pid"
				exist, _ := FileExists(pidFile)
				if exist {
					os.Remove(pidFile)
				}

				os.Exit(1)
			}
		}
	}()

	// 初始化统计管理器
	statManager.init(appDir)

	// 加载应用配置
	appManager.loadAppConfig()

	address := fmt.Sprintf("%s:%d", appConfig.Host, appConfig.Port)
	log.Printf("start %s:%d\n", appConfig.Host, appConfig.Port)

	// 初始化Handler管理器
	handlerManager.init()

	// 加载数据
	appManager.reload()

	// 启动Server
	go func() {
		err = nil
		if len(appConfig.SSL.Key) == 0 || len(appConfig.SSL.Cert) == 0 {
			err = http.ListenAndServe(address, serverMux)
		} else {
			err = http.ListenAndServeTLS(address, appConfig.SSL.Cert, appConfig.SSL.Key, serverMux)
		}

		// 处理错误
		const escape = "\x1b"
		if err != nil {
			if appManager.IsDebug {
				log.Fatal(fmt.Sprintf("%s[1;31mFailed to start app server, error:"+err.Error()+"%s[0m", escape, escape))
			} else {
				log.Fatal("Failed to start app server, error:" + err.Error())
			}
		}
	}()

	// 启动Admin
	adminManager.Load(appDir)

	// 启动缓存
	cacheManager.init()

	// 启动插件
	pluginManager.Init()
	pluginManager.Start(appDir)

	// 等待请求
	appManager.wait()
}

// 取得App管理器
func GetAppManager() *AppManager {
	return &appManager
}

// 取得插件管理器
func GetPluginManager() *PluginManager {
	return &pluginManager
}

// 取得钩子管理器
func GetHookManager() *HookManager {
	return &hookManager
}

// 判断是否为命令
func (manager *AppManager) isCommand() (isCommand bool) {
	isCommand = true

	if len(os.Args) > 1 {
		command := os.Args[1]
		if command == "start" {
			manager.startCommand()
			return
		} else if command == "stop" {
			manager.stopCommand()
			return
		} else if command == "restart" {
			manager.restartCommand()
			return
		} else if command == "reload" {
			manager.reloadCommand()
			return
		} else if command == "debug" {
			manager.IsDebug = true
			isCommand = false
			return
		} else if command == "help" || command == "-h" || command == "-help" || command == "--help" {
			manager.helpCommand()
			return
		} else if command == "version" || command == "-v" {
			manager.versionCommand()
			return
		} else if command == "create" {
			manager.createCommand()
			return
		}

		isCommand = false
		log.Println("unsupported args '" + strings.Join(os.Args[1:], " ") + "'")
		return
	}

	isCommand = false

	return
}

// 在后端运行
func (manager *AppManager) startCommand() {
	// 是否已经有进程
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
func (manager *AppManager) stopCommand() {
	log.Println("stopping the server ...")

	process, err := manager.findRunningProcess()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(1 * time.Second)

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
func (manager *AppManager) restartCommand() {
	manager.stopCommand()
	time.Sleep(time.Microsecond * 100)
	manager.startCommand()
}

// 重新加载Api
func (manager *AppManager) reloadCommand() {
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
func (manager *AppManager) helpCommand() {
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

  ./meloy-api create [API File Name]
  	Create new api, such as: ./meloy-api create hello_world_test

  ./meloy-api version
  	Show api version

  ./meloy-api help
  	Show this help`)

}

// 打印版本信息
func (manager *AppManager) versionCommand() {
	fmt.Println("  MeloyAPI v" + MELOY_API_VERSION)
	fmt.Println("  GitHub: https://github.com/iwind/MeloyApi")
	fmt.Println("  Author: Liu Xiang Chao")
	fmt.Println("  QQ: 19644627")
	fmt.Println("  E-mail: 19644627@qq.com")
}

// 创建API
func (manager *AppManager) createCommand() {
	if len(os.Args) <= 2 {
		fmt.Print("Usage: ./meloy-api create [API Code]\n\n")
		return
	}

	serverCode := "代号"
	servers := manager.loadServers()
	if len(servers) > 0 {
		serverCode = servers[0].Code
	}

	code := os.Args[2]
	apiFile := manager.AppDir + "/apis/" + code + ".json"
	err := ioutil.WriteFile(apiFile, []byte(`{
  "path": "/`+ strings.Replace(code, "_", "/", -1)+ `",
  "name": "接口名称",
  "description": "接口描述",
  "address": "%{server.`+ serverCode+ `}%{api.path}",
  "methods": [ "get", "post" ],
  "params": [
    {
      "name": "",
      "type": "",
      "description": ""
    }
  ]
}`), 0777)
	if err != nil {
		fmt.Println("Error:" + err.Error())
	} else {
		os.Chmod(apiFile, 0777)
		fmt.Println("'apis/" + code + ".json' created")
	}
}

// 加载App配置
func (manager *AppManager) loadAppConfig() {
	appBytes, appErr := ioutil.ReadFile(manager.AppDir + "/config/app.json")
	if appErr != nil {
		log.Printf("Error:%s\n", appErr)
		return
	}

	// 删除注释
	jsonString := string(appBytes)
	commentReg, err := ReuseRegexpCompile("/([*]+((.|\n|\r)+?)[*]+/)|(\n\\s+//.+)")
	if err != nil {
		log.Printf("Error:%s\n", err)
	} else {
		jsonString = commentReg.ReplaceAllString(jsonString, "")
	}

	jsonError := json.Unmarshal([]byte(jsonString), &appConfig)
	if jsonError != nil {
		log.Printf("Error:%s", jsonError)
		return
	}

	// 用户限制
	appConfig.hasUsers = len(appConfig.Users) > 0

	// 客户端限制
	appConfig.hasAllow = len(appConfig.Allow.Clients) > 0
	appConfig.hasDeny = len(appConfig.Deny.Clients) > 0

	// 请求限制
	appConfig.limitDayLeft = appConfig.Limits.Requests.Day
	appConfig.limitMinuteLeft = appConfig.Limits.Requests.Minute
	appConfig.hasDayLimit = appConfig.limitDayLeft > 0
	appConfig.hasMinuteLimit = appConfig.limitMinuteLeft > 0
}

// 重新加载API配置
func (manager *AppManager) reload() {
	// 应用配置
	manager.loadAppConfig()

	// 服务器配置
	servers := appManager.loadServers()
	ApiArray = []Api{}
	appManager.loadApis(manager.AppDir+string(os.PathSeparator)+"apis", servers, &ApiArray)

	handlerManager.disableAll()

	//处理pattern
	if !serverMuxLoaded {
		serverMuxLoaded = true

		serverMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			// @TODO 需要更好的性能
			for _, api := range ApiArray {
				if !api.IsEnabled {
					continue
				}
				if len(api.patternNames) > 0 && api.patternRegexp.MatchString(request.URL.Path) {
					matches := api.patternRegexp.FindStringSubmatch(request.URL.Path)

					values := url.Values{}
					for index, name := range api.patternNames {
						values.Add(name, matches[index+1])
					}

					if len(request.URL.RawQuery) == 0 {
						request.URL.RawQuery = values.Encode()
					} else {
						request.URL.RawQuery += "&" + values.Encode()
					}

					request.URL.Path = api.Path
					break
				}
			}
			request.RequestURI = request.URL.Path + "?" + request.URL.Query().Encode()
			handlerManager.handle(writer, request)
		})
	}

	// 处理路径
	for _, api := range ApiArray {
		log.Println("load api '" + api.Path + "' from '" + strings.TrimPrefix(api.File, manager.AppDir+string(os.PathSeparator)+"apis"+string(os.PathSeparator)) + "'")

		if !api.IsEnabled {
			continue
		}

		func(api Api) {
			handler, ok := handlerManager.find(api.Path)
			if ok {
				handler.isEnabled = true
				handler.Api.copyFrom(api)
			} else {
				handlerManager.HandleFunc(serverMux, &api, func(writer http.ResponseWriter, request *http.Request) {
					appManager.handle(writer, request, &api)
				})
			}
		}(api)
	}

	// 刷新管理数据
	adminManager.Reload()

	// 刷新插件
	pluginManager.Reload()
}

// 等待处理请求
func (manager *AppManager) wait() {
	defer statManager.closeDb()

	// Hold住进程
	var wg = sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

// 加载服务器列表
func (manager *AppManager) loadServers() (servers []Server) {
	serverBytes, serverErr := ioutil.ReadFile(manager.AppDir + "/config/servers.json")

	if serverErr != nil {
		log.Printf("Error:%s\n", serverErr)
		return
	}

	// 删除注释
	jsonString := string(serverBytes)
	commentReg, err := ReuseRegexpCompile("/([*]+((.|\n|\r)+?)[*]+/)|(\n\\s+//.+)")
	if err != nil {
		log.Printf("Error:%s\n", err)
	} else {
		jsonString = commentReg.ReplaceAllString(jsonString, "")
	}

	jsonErr := json.Unmarshal([]byte(jsonString), &servers)
	if jsonErr != nil {
		log.Printf("Error:%s:\n~~~\n%s\n~~~\n", jsonErr, jsonString)
		return
	}

	// 分析Servers
	for index, server := range servers {
		// 分析最大尺寸
		if len(server.Request.MaxSize) > 0 {
			size, err := parseSizeFromString(server.Request.MaxSize)
			if err != nil {
				log.Println("Parse "+server.Request.MaxSize+" Error:", err.Error())
			} else {
				servers[index].Request.maxSizeBits = size
			}
		}

		// 分析超时时间
		if len(server.Request.Timeout) > 0 {
			reg, _ := regexp.Compile("^(\\d+(?:\\.\\d+)?)\\s*(ms|s)$")
			matches := reg.FindStringSubmatch(server.Request.Timeout)
			if len(matches) == 3 {
				duration, err := time.ParseDuration(matches[1] + matches[2])
				if err != nil {
					log.Println("API timeout parse failed '" + server.Request.Timeout + "'")
				}
				servers[index].Request.timeoutDuration = duration
			} else {
				log.Println("API timeout parse failed '" + server.Request.Timeout + "'")
			}
		}
	}

	return
}

// 加载Api列表
func (manager *AppManager) loadApis(apiDir string, servers []Server, apis *[]Api) {
	files, err := ioutil.ReadDir(apiDir)
	if err != nil {
		log.Printf("Error:%s\n", err)
		return
	}

	reg, err := ReuseRegexpCompile("\\.json$")
	if err != nil {
		log.Printf("Error:%s\n", err)
		return
	}

	dataReg, err := ReuseRegexpCompile("\\.mock\\.json")
	if err != nil {
		log.Printf("Error:%s\n", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			manager.loadApis(apiDir+string(os.PathSeparator)+file.Name(), servers, apis)
			continue
		}

		// 跳过假数据
		if dataReg.MatchString(file.Name()) {
			continue
		}

		if !reg.MatchString(file.Name()) {
			continue
		}

		_bytes, err := ioutil.ReadFile(apiDir + string(os.PathSeparator) + file.Name())
		if err != nil {
			log.Printf("Error:%s:%s\n", file.Name(), err)
			continue
		}

		var api = Api{
			IsEnabled: true,
		}

		jsonString := string(_bytes)

		// 删除注释
		commentReg, err := ReuseRegexpCompile("/([*]+((.|\n|\r)+?)[*]+/)|(\n\\s+//.+)")
		if err != nil {
			continue
		}
		jsonString = commentReg.ReplaceAllString(jsonString, "")

		jsonError := json.Unmarshal([]byte(jsonString), &api)
		if jsonError != nil {
			log.Printf("Error:%s:%s:\n~~~\n%s\n~~~\n", file.Name(), jsonError.Error(), jsonString)
			continue
		}
		api.File = apiDir + string(os.PathSeparator) + file.Name()
		api.parse()

		// 转换地址
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

			// 支持变量 %{server.服务器代号}, %{api.path}
			reg, _ := ReuseRegexpCompile("%{server." + server.Code + "}")
			pathReg, _ := ReuseRegexpCompile("%\\{api.path}")

			if !reg.MatchString(api.Address) {
				continue
			}

			// 超时和最大请求尺寸设置
			if api.maxSizeBits <= 0 && server.Request.maxSizeBits > 0 {
				api.maxSizeBits = server.Request.maxSizeBits
			}
			if api.maxSizeBits <= 0 { // 如果没有设置，默认只支持32M
				api.maxSizeBits = 32 << 20
			}
			if api.timeoutDuration <= 0 && server.Request.timeoutDuration > 0 {
				api.timeoutDuration = server.Request.timeoutDuration
			}

			for _, host := range server.Hosts {
				address := reg.ReplaceAllString(api.Address, host.Address)
				address = pathReg.ReplaceAllString(address, api.Path)

				if totalWeight == 0 {
					api.Addresses = append(api.Addresses, ApiAddress{
						Server: server.Code,
						Host:   host.Address,
						URL:    address,
					})
					continue
				}

				weight := int(host.Weight * 10 / totalWeight)
				for i := 0; i < weight; i++ {
					api.Addresses = append(api.Addresses, ApiAddress{
						Server: server.Code,
						Host:   host.Address,
						URL:    address,
					})
				}
			}
		}

		// 假数据
		fileName := file.Name()
		reg, _ := ReuseRegexpCompile("\\.json")
		dataFileName := apiDir + string(os.PathSeparator) + reg.ReplaceAllString(fileName, ".mock.json")

		fileExists, _ := FileExists(dataFileName)
		if fileExists {
			_bytes, err := ioutil.ReadFile(dataFileName)
			if err != nil {
				log.Println("Error:" + err.Error())
			} else {
				api.Mock = string(_bytes)
			}
		}

		api.countAddresses = len(api.Addresses)
		*apis = append(*apis, api)
	}

	return
}

// 处理请求
func (manager *AppManager) handle(writer http.ResponseWriter, request *http.Request, api *Api) {
	// 登录用户
	if !manager.validateUser(request) {
		http.Error(writer, "Permission Denied", http.StatusForbidden)
		return
	}

	// 校验请求
	if !manager.validateRequest(request) {
		http.Error(writer, "Permission Denied", http.StatusForbidden)
		return
	}

	// 处理限流
	if manager.reachLimit() {
		http.Error(writer, "API requests limit reached", http.StatusForbidden)
		return
	}

	if api.countAddresses == 0 {
		fmt.Fprintln(writer, "Does not have avaiable address")
		return
	}

	// 选取地址
	var address ApiAddress
	if api.countAddresses > 1 {
		rand.Seed(time.Now().UnixNano())
		index := rand.Int() % api.countAddresses
		address = api.Addresses[index]
	} else {
		address = api.Addresses[0]
	}

	// 检查method
	method := strings.ToUpper(request.Method)
	if !containsString(api.Methods, method) {
		fmt.Fprintln(writer, "'"+request.Method+"' method is not supported")
		return
	}

	hookManager.beforeHook(writer, request, api, func(hookContext *HookContext) {
		if api.IsAsynchronous {
			manager.setApiHeaders(writer, api)
			writer.Write([]byte(api.responseString))
			go manager.handleMethod(writer, request, api, address, method, hookContext)
		} else {
			// 开始处理
			manager.handleMethod(writer, request, api, address, method, hookContext)
		}
	})
}

// 转发某个方法的请求
func (manager *AppManager) handleMethod(writer http.ResponseWriter, request *http.Request, api *Api, address ApiAddress, method string, hookContext *HookContext) {
	t := time.Now().UnixNano()

	query := request.URL.RawQuery

	// 判断最大内容长度
	if api.maxSizeBits > 0 && float64(request.ContentLength) > api.maxSizeBits {
		request.ParseMultipartForm(2 << 10)
		http.Error(writer, "request body too large to upload", http.StatusRequestEntityTooLarge)
		return
	}

	// 是否有缓存
	cacheKey := request.URL.RequestURI()
	cacheEntry, ok := cacheManager.get(cacheKey)
	if ok {
		for key, values := range cacheEntry.Header {
			for _, value := range values {
				writer.Header().Add(key, value)
			}
		}

		manager.setApiHeaders(writer, api)
		writer.Write(cacheEntry.Bytes)

		statManager.send(address, api.Path, request.RequestURI, (time.Now().UnixNano()-t)/1000000, 0, 1)

		return
	}

	requestURL := address.URL
	uri := request.RequestURI
	if len(query) > 0 {
		requestURL += "?" + query
	}

	newRequest, err := http.NewRequest(method, requestURL, nil)

	if err != nil {
		log.Println("Error:" + err.Error())

		manager.setApiHeaders(writer, api)

		hookManager.afterHook(hookContext, nil, err)
		statManager.send(address, api.Path, request.RequestURI, (time.Now().UnixNano()-t)/1000000, 1, 0)
		return
	}

	newRequest.Header = request.Header
	request.Header.Set("Meloy-Api", "1.0")
	newRequest.Body = request.Body

	// 超时时间
	if api.timeoutDuration > 0 {
		requestClient.Timeout = api.timeoutDuration
	} else {
		requestClient.Timeout = 30 * time.Second
	}

	// 是否正在watch
	var isWatching = false
	if appConfig.isWatching {
		if appConfig.watchingAt > time.Now().Unix()-60 {
			isWatching = true
		} else {
			appConfig.isWatching = false
		}
	}

	var requestCopy *http.Request
	if isWatching {
		if request.ContentLength > 65535 {
			requestCopy, _ = http.NewRequest(request.Method, request.RequestURI, ioutil.NopCloser(bytes.NewReader([]byte{})))
			requestCopy.ContentLength = request.ContentLength
			requestCopy.RequestURI = request.RequestURI
			requestCopy.Header = request.Header
		} else {
			buf, err := ioutil.ReadAll(request.Body)

			if err == nil {
				requestBodyCopy := ioutil.NopCloser(bytes.NewBuffer(buf))
				requestBodyCopy2 := ioutil.NopCloser(bytes.NewBuffer(buf))
				requestCopy, _ = http.NewRequest(request.Method, request.RequestURI, requestBodyCopy)
				requestCopy.ContentLength = request.ContentLength
				requestCopy.RequestURI = request.RequestURI
				requestCopy.Header = request.Header
				newRequest.Body = requestBodyCopy2
				defer requestBodyCopy.Close()
				defer requestBodyCopy2.Close()
			} else {
				log.Println("Error:" + err.Error())
			}
		}

	}

	resp, err := requestClient.Do(newRequest)

	if err != nil {
		manager.setApiHeaders(writer, api)

		log.Println("Error:" + err.Error())
		hookManager.afterHook(hookContext, nil, err)

		// 统计
		statManager.send(address, api.Path, request.RequestURI, (time.Now().UnixNano()-t)/1000000, 1, 0)
		return
	}

	// 监控日志
	if isWatching {
		if requestCopy != nil {
			statManager.sendRequest(resp, requestCopy)
		}
	}

	// 调用钩子
	hookManager.afterHook(hookContext, resp, nil)

	// 分析头部指令等信息
	apiConfig := ApiConfig{
		cacheTags: []string{"$MeloyAPI$" + api.Path},
	}
	manager.parseResponseHeaders(writer, request, resp, address, api, &apiConfig)
	manager.setApiHeaders(writer, api)

	_bytes := []byte{}
	if api.hasResponseString {
		err = nil
		_bytes = []byte(api.responseString)
	} else {
		_bytes, err = ioutil.ReadAll(resp.Body)
	}

	resp.Body.Close()

	if err != nil {
		log.Println("Error:" + err.Error())
		statManager.send(address, api.Path, uri, (time.Now().UnixNano()-t)/1000000, 1, 0)
		return
	}

	// 缓存
	if apiConfig.cacheLifeMs > 0 {
		cacheManager.set(cacheKey, apiConfig.cacheTags, _bytes, writer.Header(), apiConfig.cacheLifeMs)
	}

	// 如果不是异步请求的，就返回请求得到的数据
	if !api.IsAsynchronous {
		writer.Write(_bytes)
	}

	var errors int64 = 0
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errors++
		log.Println("Error: api return ", resp.Status)
	}

	statManager.send(address, api.Path, uri, (time.Now().UnixNano()-t)/1000000, errors, 0)
}

// 分析响应头部
func (manager *AppManager) parseResponseHeaders(writer http.ResponseWriter, request *http.Request, resp *http.Response, address ApiAddress, api *Api, apiConfig *ApiConfig) {
	directiveReg, _ := ReuseRegexpCompile("^Meloy-Api-(.+)")

	for key, values := range resp.Header {
		if containsString([]string{"Connection", "Server"}, key) {
			continue
		}

		if api.hasResponseString && key == "Content-Encoding" {
			continue
		}

		// 处理指令
		if directiveReg.MatchString(key) {
			directive := directiveReg.FindStringSubmatch(key)[1]
			manager.processDirective(request, address, api.Path, directive, values[0], apiConfig)

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
	// 调试信息
	{
		reg, _ := ReuseRegexpCompile("^Debug")
		if reg.MatchString(directive) {
			statManager.sendDebug(address, path, request.URL.RequestURI(), value)
			return
		}
	}

	// 设置缓存时间
	{
		reg, _ := ReuseRegexpCompile("^Cache-Life-Ms")
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

	// 设置缓存标签
	{
		reg, _ := ReuseRegexpCompile("^Cache-Tag")
		if reg.MatchString(directive) {
			if apiConfig.cacheTags == nil {
				apiConfig.cacheTags = []string{}
			}
			apiConfig.cacheTags = append(apiConfig.cacheTags, value)

			return
		}
	}

	// 设置要删除的缓存标签
	{
		reg, _ := ReuseRegexpCompile("^Cache-Delete")
		if reg.MatchString(directive) {
			cacheManager.deleteTag(value)

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

// 查找正在运行的MeloyAPI进程
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

// 设置是否监听
func (manager *AppManager) setWatching(isWatching bool) {
	appConfig.isWatching = isWatching
	appConfig.watchingAt = time.Now().Unix()
}

// 校验用户
func (manager *AppManager) validateUser(request *http.Request) bool {
	if !appConfig.hasUsers {
		return true
	}

	username := request.Header.Get("Meloy-Username")
	password := request.Header.Get("Meloy-Password")

	if len(username) == 0 || len(password) == 0 {
		return false
	}

	for _, config := range appConfig.Users {
		if config.Type == "account" {
			if config.Username == username && config.Password == password {
				return true
			}
		}
	}

	return false
}

// 校验请求
func (manager *AppManager) validateRequest(request *http.Request) bool {
	if !appConfig.hasAllow && !appConfig.hasDeny {
		return true
	}

	reg, _ := ReuseRegexpCompile(":\\d+$")
	ip := reg.ReplaceAllString(request.RemoteAddr, "")

	// 本地的
	if appConfig.Host == "0.0.0.0" && ip == "[::1]" {
		return true
	}

	// 禁止的
	if appConfig.hasDeny {
		if containsString(appConfig.Deny.Clients, ip) {
			return false
		}
	}

	// 支持的
	if appConfig.hasAllow {
		if !containsString(appConfig.Allow.Clients, ip) {
			return false
		}
	}

	return true
}

// 判断是否达到请求限制
func (manager *AppManager) reachLimit() bool {
	if !appConfig.hasMinuteLimit && !appConfig.hasDayLimit {
		return false
	}

	now := time.Now()

	if appConfig.hasMinuteLimit {
		currentMinute := fmt.Sprintf("%04d-%02d-%02d %02d:%02d", now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute())
		if currentMinute != appConfig.limitLastMinute {
			appConfig.limitLastMinute = currentMinute
			appConfig.limitMinuteLeft = appConfig.Limits.Requests.Minute
		}

		if appConfig.limitMinuteLeft <= 0 {
			return true
		}
		appConfig.limitMinuteLeft--
	}

	if appConfig.hasDayLimit {
		currentDay := fmt.Sprintf("%04d-%02d-%02d", now.Year(), int(now.Month()), now.Day())
		if currentDay != appConfig.limitLastDay {
			appConfig.limitLastDay = currentDay
			appConfig.limitDayLeft = appConfig.Limits.Requests.Day
		}

		if appConfig.limitDayLeft <= 0 {
			return true
		}
		appConfig.limitDayLeft--
	}

	return false
}

// 设置API头部信息
func (manager *AppManager) setApiHeaders(writer http.ResponseWriter, api *Api) {
	// 写入Headers
	if len(api.Headers) == 0 {
		return
	}

	for _, header := range api.Headers {
		writer.Header().Set(header.Name, header.Value)
	}
}
