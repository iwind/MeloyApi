package MeloyApi

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"log"
	"fmt"
	"regexp"
	"strconv"
	"os/exec"
	"os"
	"time"
	"runtime"
	"strings"
)

type AdminManager struct {

}

type AdminConfig struct {
	Host string
	Port int
	SSL struct {
		Cert string
		Key string
	}

	Allow struct {
		Clients []string
	}
	Deny struct {
		Clients []string
	}

	hasAllow bool
	hasDeny bool
}

var adminConfig AdminConfig
var adminApiMapping map[string] Api

// 加载Admin
func (manager *AdminManager)Load(appDir string)  {
	bytes, err := ioutil.ReadFile(appDir + "/config/admin.json")
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	// 删除注释
	jsonString := string(bytes)
	commentReg, err := ReuseRegexpCompile("/([*]+((.|\n|\r)+?)[*]+/)|(\n\\s+//.+)")
	if err != nil {
		log.Printf("Error:%s\n", err)
	} else {
		jsonString = commentReg.ReplaceAllString(jsonString, "")
	}

	err = json.Unmarshal([]byte(jsonString), &adminConfig)
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	adminConfig.hasAllow = len(adminConfig.Allow.Clients) > 0
	adminConfig.hasDeny = len(adminConfig.Deny.Clients) > 0

	address := fmt.Sprintf("%s:%d", adminConfig.Host, adminConfig.Port)
	log.Println("start " + address)

	manager.Reload()

	go func() {
		serverMux := http.NewServeMux()
		serverMux.HandleFunc("/", manager.handleRequest)
		serverMux.Handle("/web/", http.StripPrefix("/web/",
				http.FileServer(http.Dir(appDir + string(os.PathSeparator) + "web" + string(os.PathSeparator)))))

		err = nil
		if len(adminConfig.SSL.Key) == 0 || len(adminConfig.SSL.Cert) == 0 {
			err = http.ListenAndServe(address, serverMux)
		} else {
			err = http.ListenAndServeTLS(address, adminConfig.SSL.Cert, adminConfig.SSL.Key, serverMux)
		}

		// 处理错误
		const escape = "\x1b"
		if err != nil {
			if appManager.IsDebug {
				log.Fatal(fmt.Sprintf("%s[1;31mFailed to start admin server, error:" + err.Error() + "%s[0m", escape, escape))
			} else {
				log.Fatal("Failed to start admin server, error:" + err.Error())
			}
		}
	}()
}

// 重新加载数据
func (manager *AdminManager)Reload() {
	adminApiMapping = make(map [string] Api)
	for _, api := range ApiArray {
		adminApiMapping[api.Path] = api //@TODO 支持  /:name/:age => /abc/123
	}
}

// 处理请求
func (manager *AdminManager)handleRequest(writer http.ResponseWriter, request *http.Request)  {
	request.ParseForm()
	if request.Form.Get("_editor") == "true" {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	if !manager.validateRequest(writer, request) {
		return
	}

	path := request.URL.Path

	if path == "/" {
		writer.Header().Set("Location", "/web/")
		writer.WriteHeader(301)
		return
	}

	if path == "/@api/all" {
		manager.handleApis(writer, request)
		return
	}

	if path == "/@api/reload" {
		manager.handleReloadApis(writer, request)
		return
	}

	{
		reg, _ := regexp.Compile("^/@mock(/.+)$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleMock(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleApi(writer, request, matches[1])
			return
		}
	}

	// 删除API
	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/delete$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleApiDelete(writer, request, matches[1])
			return
		}
	}

	// 更改API
	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/update$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleApiUpdate(writer, request, matches[1])
			return
		}
	}

	// 更改API文件名称
	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/rename/(.+)$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleApiRename(writer, request, matches[1], matches[2])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/year/(\\d+)/month/(\\d+)/day/(\\d+)$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			year, _ := strconv.Atoi(matches[2])
			month, _ := strconv.Atoi(matches[3])
			day, _ := strconv.Atoi(matches[4])
			manager.handleApiDay(writer, request, matches[1], year, month, day)
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/debug/logs$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleDebugLogs(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/debug/flush$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleDebugFlush(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/?$")
		if reg.MatchString(path) {
			manager.handleIndex(writer, request)

			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/clear$")
		if reg.MatchString(path) {
			manager.handleCacheClear(writer, request)
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/\\[(.+)]/clear$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleCacheClearPath(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/tag/(.+)/delete$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleCacheDeleteTag(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/tag/(.+)")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleCacheTagInfo(writer, request, matches[1])
			return
		}
	}

	if path == "/@git/pull" {
		manager.handleGitPull(writer, request)
		return
	}

	if path == "/@monitor" {
		manager.handleMonitor(writer, request)
		return
	}

	if path == "/@api/stat" {
		manager.handleStat(writer, request)
		return
	}

	if path == "/@api/stat/requests/rank" {
		manager.handleStatRequestsRank(writer, request)
		return
	}

	if path == "/@api/stat/hits/rank" {
		manager.handleStatHitsRank(writer, request)
		return
	}

	if path == "/@api/stat/errors/rank" {
		manager.handleStatErrorsRank(writer, request)
		return
	}

	if path == "/@api/stat/cost/rank" {
		manager.handleStatCostRank(writer, request)
		return
	}

	if path == "/@api/watch" {
		manager.handleWatch(writer, request)
		return 
	}

	if path == "/@api/watch/clear" {
		manager.handleWatchClear(writer, request)
		return
	}

	{
		fmt.Fprint(writer, "404 page not found (" + path + ")")
	}
}

// /@api
// 处理API根目录请求
func (manager *AdminManager)handleIndex(writer http.ResponseWriter, request *http.Request) {
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"version": MELOY_API_VERSION,
			"goVersion": runtime.Version(),
			"goRoot": runtime.GOROOT(),
		},
	})
}

// /@mock/:path
// 输出模拟数据
func (manager *AdminManager)handleMock(writer http.ResponseWriter, _ *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if ok && len(api.Mock) > 0 {
		//删除注释以避免JSON解析错误
		mock := api.Mock
		reg, _ := regexp.Compile("\n\\s*//.+")
		mock = reg.ReplaceAllString(mock, "")

		fmt.Fprint(writer, mock)
	} else {
		writer.Write([]byte("404 page not found (" + path + ")"))
	}
}

// /@api/[:path]
// 输出某个API信息
func (manager *AdminManager)handleApi(writer http.ResponseWriter, request *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if !ok {
		manager.printJSON(writer, request, Map {
			"code": 404,
			"message": "Not Found",
			"data": Api{},
		})
	} else {
		manager.printJSON(writer, request, Map {
			"code": 200,
			"message": "Success",
			"data": api,
		})
	}
}

// /@api/path/year/:year/month/:month/day/:day
// 日统计
func (manager *AdminManager)handleApiDay(writer http.ResponseWriter, request *http.Request, path string, year int, month int, day int) {
	apiStat := statManager.findAvgStatForDay(path, year, month, day)
	minutes := statManager.findMinuteStatForDay(path, year, month, day)

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"avgMs": apiStat.AvgMs,
			"requests": apiStat.Requests,
			"hits": apiStat.Hits,
			"errors": apiStat.Errors,
			"minutes": minutes,
		},
	})
}

// /@api/[:path]/delete
// 删除API
func (manager *AdminManager)handleApiDelete(writer http.ResponseWriter, request *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if !ok {
		manager.printJSON(writer, request, Map {
			"code": 404,
			"message": "Not found",
			"data": nil,
		})
	} else {
		now := time.Now()
		timeString := fmt.Sprintf("%d%02d%02d_%02d%02d%02d", now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())
		newFile := api.File + "." + timeString + ".deleted"
		os.Rename(api.File, newFile)
		appManager.reload()

		manager.printJSON(writer, request, Map {
			"code": 200,
			"message": "Success",
			"data": newFile,
		})
	}
}

// /@api/[:path]/update
// 更改API信息
func (manager *AdminManager)handleApiUpdate(writer http.ResponseWriter, request *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if !ok {
		manager.printJSON(writer, request, Map {
			"code": 404,
			"message": "Not found",
			"data": nil,
		})
	} else {
		newBytes, err := ioutil.ReadAll(request.Body)
		if err != nil {
			manager.printJSON(writer, request, Map {
				"code": 500,
				"message": err.Error(),
				"data": nil,
			})
			return
		}
		log.Println(api.Path, string(newBytes))

		ioutil.WriteFile(api.File, newBytes, 0777)

		appManager.reload()

		manager.printJSON(writer, request, Map {
			"code": 200,
			"message": "Success",
			"data": nil,
		})
	}
}

// /@api/[:path]/rename
// 更改API文件名称
func (manager *AdminManager)handleApiRename(writer http.ResponseWriter, request *http.Request, path string, toFile string) {
	api, ok := adminApiMapping[path]
	if !ok {
		manager.printJSON(writer, request, Map {
			"code": 404,
			"message": "Not found",
			"data": nil,
		})
	} else {
		newFile := appManager.AppDir + string(os.PathSeparator) + "apis" + string(os.PathSeparator) + toFile
		os.Rename(api.File, newFile)
		appManager.reload()

		manager.printJSON(writer, request, Map {
			"code": 200,
			"message": "Success",
			"data": newFile,
		})
	}
}

// /@api/[:path]/debug/logs
// 打印调试日志
func (manager *AdminManager)handleDebugLogs(writer http.ResponseWriter, request *http.Request, path string) {
	logs := statManager.findDebugLogsForPath(path)
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": len(logs),
			"logs": logs,
		},
	})
}

// /@api/[:path]/debug/flush
// 刷新调试日志
func (manager *AdminManager)handleDebugFlush(writer http.ResponseWriter, request *http.Request, _ string) {
	err, count := statManager.flushDebugLogs()
	if err != nil {
		manager.printJSON(writer, request, Map {
			"code": 500,
			"message": err.Error(),
			"data": Map {
				"count": count,
			},
		})
		return
	}

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
}

// /@api/all
// 输出所有API信息
func (manager *AdminManager)handleApis(writer http.ResponseWriter, request *http.Request) {
	//统计相关
	var arr = ApiArray
	for index, api := range arr {
		api.Stat = statManager.avgStat(api.Path)
		arr[index] = api
	}

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": ApiArray,
	})
}

// /@api/reload
// 刷新API配置
func (manager *AdminManager)handleReloadApis(writer http.ResponseWriter, request *http.Request) {
	appManager.reload()

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": nil,
	})
}

// /@cache/clear
// 清除所有缓存
func (manager *AdminManager)handleCacheClear(writer http.ResponseWriter, request *http.Request) {
	count := cacheManager.clearAll()

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
}

// /@cache/[:path]/clear
// 清除某个API对应的所有Cache
func (manager *AdminManager)handleCacheClearPath(writer http.ResponseWriter, request *http.Request, path string)  {
	count := cacheManager.deleteTag("$MeloyAPI$" + path)

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
}

// /@cache/tag/:tag/delete
// 删除某个标签对应的缓存
func (manager *AdminManager)handleCacheDeleteTag(writer http.ResponseWriter, request *http.Request, tag string) {
	count := cacheManager.deleteTag(tag)

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
}

// /@cache/tag/:tag
// 打印某个标签信息
func (manager *AdminManager)handleCacheTagInfo(writer http.ResponseWriter, request *http.Request, tag string) {
	count, keys, ok := cacheManager.statTag(tag)
	if !ok {
		manager.printJSON(writer, request, Map {
			"code": 404,
			"message": "Not found",
			"data": Map {
				"count": count,
				"keys": keys,
			},
		})
	} else {
		manager.printJSON(writer, request, Map {
			"code": 200,
			"message": "Success",
			"data": Map {
				"count": count,
				"keys": keys,
			},
		})
	}
}

// /@git/pull
// 处理Git Pull命令
func (manager *AdminManager)handleGitPull(writer http.ResponseWriter, request *http.Request) {
	cmd := exec.Command("sh", "-c", "cd " + appManager.AppDir + ";git pull;touch /tmp/tmp-go-file")

	stdout, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		manager.writeErrorMessage(writer, request, stdoutErr)
		return
	}

	defer stdout.Close()

	runErr := cmd.Start()

	_bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		manager.writeErrorMessage(writer, request, err)
		cmd.Wait()
		return
	}
	var output = string(_bytes)

	cmd.Wait()

	if runErr != nil {
		manager.writeErrorMessage(writer, request, runErr)
		return
	}

	//刷新数据
	go appManager.reload()

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": output,
		"data": nil,
	})
}

// /@monitor
// 监控信息
func (manager *AdminManager)handleMonitor(writer http.ResponseWriter, request *http.Request) {
	//内存信息
	memoryStat := runtime.MemStats{}
	runtime.ReadMemStats(&memoryStat)

	//负载信息
	load1 := "0"
	load2 := "0"
	load3 := "0"
	func () {
		cmd := exec.Command("uptime")
		reader, readerErr := cmd.StdoutPipe()
		if readerErr != nil {
			return
		}

		defer reader.Close()

		runErr := cmd.Start()
		if runErr != nil {
			return
		}

		bytes, err := ioutil.ReadAll(reader)
		if err != nil {
			return
		}
		cmd.Process.Wait()

		resultString := string(bytes)
		resultString = strings.Replace(resultString, ",", " ", -1)
		resultString = strings.Replace(resultString, ";", " ", -1)
		reg, _ := regexp.Compile("load average(?:s)?\\s*:\\s*(\\S+)\\s*(\\S+)\\s*(\\S+)")
		matches := reg.FindStringSubmatch(resultString)
		if len(matches) > 0 {
			load1Float, _ := strconv.ParseFloat(matches[1], 32)
			load2Float, _ := strconv.ParseFloat(matches[2], 32)
			load3Float, _ := strconv.ParseFloat(matches[3], 32)
			load1 = fmt.Sprintf("%.2f", load1Float)
			load2 = fmt.Sprintf("%.2f", load2Float)
			load3 = fmt.Sprintf("%.2f", load3Float)
		}
	}()

	stat, _ := statManager.findStat()

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"memory": memoryStat.Sys,
			"memoryFormatted": fmt.Sprintf("%.2fm", float64(memoryStat.Sys) / 1024 / 1024),
			"heap": memoryStat.HeapSys,
			"heapFormatted": fmt.Sprintf("%.2fm", float64(memoryStat.HeapSys) / 1024 / 1024),
			"heapObjects": memoryStat.HeapObjects,
			"routines": runtime.NumGoroutine(),
			"load1m": load1,
			"load5m": load2,
			"load15m": load3,
			"requestsPerMin": stat["requests"],
			"hitsPercent": stat["hits"],
			"errorsPercent": stat["errors"],
			"cost": stat["ms"],
		},
	})
}

// /@api/stat
// 总体统计
func (manager *AdminManager) handleStat(writer http.ResponseWriter, request *http.Request) {
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": statManager.findGlobalStat(),
	})
}

// /@api/stat/requests/rank
// 请求数排行
func (manager *AdminManager) handleStatRequestsRank(writer http.ResponseWriter, request *http.Request) {
	apis, err := statManager.findRequestsRank(10)
	if err != nil {
		manager.writeErrorMessage(writer, request, err)
		return
	}
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": apis,
	})
}


// /@api/stat/hits/rank
// 缓存命中数排行
func (manager *AdminManager) handleStatHitsRank(writer http.ResponseWriter, request *http.Request) {
	apis, err := statManager.findHitsRank(10)
	if err != nil {
		manager.writeErrorMessage(writer, request, err)
		return
	}
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": apis,
	})
}


// /@api/stat/errors/rank
// 错误数排行
func (manager *AdminManager) handleStatErrorsRank(writer http.ResponseWriter, request *http.Request) {
	apis, err := statManager.findErrorsRank(10)
	if err != nil {
		manager.writeErrorMessage(writer, request, err)
		return
	}
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": apis,
	})
}

// /@api/stat/cost/rank
// 错误数排行
func (manager *AdminManager) handleStatCostRank(writer http.ResponseWriter, request *http.Request) {
	apis, err := statManager.findCostRank(10)
	if err != nil {
		manager.writeErrorMessage(writer, request, err)
		return
	}
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": apis,
	})
}

// /@api/watch
// 监控日志
func (manager *AdminManager)handleWatch(writer http.ResponseWriter, request *http.Request)  {
	appManager.setWatching(true)

	//读取日志
	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": statManager.watchLogs(),
	})
}

// /@api/watch/clear
// 清除监控日志
func (manager *AdminManager)handleWatchClear(writer http.ResponseWriter, request *http.Request)  {
	statManager.clearWatchLogs()

	manager.printJSON(writer, request, Map {
		"code": 200,
		"message": "Success",
		"data": nil,
	})
}

// 校验请求
func (manager *AdminManager)validateRequest(writer http.ResponseWriter, request *http.Request) bool {
	if !adminConfig.hasAllow && !adminConfig.hasDeny {
		return true
	}

	reg, _ := ReuseRegexpCompile(":\\d+$")
	ip := reg.ReplaceAllString(request.RemoteAddr, "")

	//本地的
	if adminConfig.Host == "0.0.0.0" && ip == "[::1]" {
		return  true
	}

	//禁止的
	if adminConfig.hasDeny {
		if containsString(adminConfig.Deny.Clients, ip) {
			manager.printJSON(writer, request, Map {
				"code": 403,
				"message": "Forbidden",
				"data": nil,
			})

			return false
		}
	}

	//支持的
	if adminConfig.hasAllow {
		if !containsString(adminConfig.Allow.Clients, ip) {
			manager.printJSON(writer, request, Map {
				"code": 403,
				"message": "Forbidden",
				"data": nil,
			})

			return false
		}
	}

	return true
}

// 写入错误信息
func (manager *AdminManager)writeErrorMessage(writer http.ResponseWriter, request *http.Request, err error) {
	manager.printJSON(writer, request, Map {
		"code": 500,
		"message": err.Error(),
		"data": nil,
	})
}

// 打印JSON信息
func (manager *AdminManager)printJSON(writer http.ResponseWriter, request *http.Request, data Map) {
	request.ParseForm()
	pretty := request.Form.Get("_pretty")
	editor := request.Form.Get("_editor")

	var bytes []byte
	var err error
	if pretty == "true" {
		bytes, err = json.MarshalIndent(data, "", "  ")

	} else {
		bytes, err = json.Marshal(data)
	}

	if err != nil {
		writer.Write([]byte("Error:" + err.Error()))
	} else {
		if editor == "true" {
			writer.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<style type="text/css">
div {
	position: absolute;
	top: 1em;
	bottom: 1em;
	right: 1em;
	left: 1em;
	margin: 0;
}

div textarea {
	border: 1px solid rgba(34,36,38,.15);
	transition: color .1s ease,border-color .1s ease;
	box-shadow: 0 0 0 0 transparent inset;
	font-size: 14px;
	line-height: 1.5;
	outline: 0;
	position: absolute;
	width: 100%;
	top: 0;
	bottom: 0;
	right: 0;
	left: 0;
	margin: 0;
	font-family: Lato,'Helvetica Neue',Arial,Helvetica,sans-serif;
}
</style>
</head>
<body>
<div><textarea>`))
		}
		writer.Write(bytes)

		if editor == "true" {
			writer.Write([]byte(`</textarea></div>
</body>
</html>
			`))
		}
	}
}