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
	"bufio"
	"io"
	"os"
	"time"
	"runtime"
)

type AdminManager struct {

}

type AdminConfig struct {
	Host string
	Port int
	Allow AdminConfigAllow
}

type AdminConfigAllow struct {
	Clients []string
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

	err = json.Unmarshal(bytes, &adminConfig)
	if err != nil {
		log.Println("Error:" + err.Error())
		return
	}

	address := fmt.Sprintf("%s:%d", adminConfig.Host, adminConfig.Port)
	log.Println("start " + address)

	manager.Reload()

	go func() {
		serverMux := http.NewServeMux()
		serverMux.HandleFunc("/", manager.handleRequest)

		http.ListenAndServe(address, serverMux)
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
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")

	if !manager.validateRequest(writer, request) {
		return
	}

	path := request.URL.Path
	if path == "/@api/all" {
		manager.handleApis(writer, request)
		return
	}

	if path == "/@api/reload" {
		manager.handleReloadApis(writer)
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
			manager.handleIndex(writer)

			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/clear$")
		if reg.MatchString(path) {
			manager.handleCacheClear(writer)
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/\\[(.+)]/clear$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleCacheClearPath(writer, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/tag/(.+)/delete$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleCacheDeleteTag(writer, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@cache/tag/(.+)")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			manager.handleCacheTagInfo(writer, matches[1])
			return
		}
	}

	if path == "/@git/pull" {
		manager.handleGitPull(writer)
		return
	}

	if path == "/@monitor" {
		manager.handleMonitor(writer)
		return
	}

	{
		fmt.Fprint(writer, "404 page not found (" + path + ")")
	}
}

// 处理API根目录请求
func (manager *AdminManager)handleIndex(writer http.ResponseWriter) {
	bytes, _ := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"version": MELOY_API_VERSION,
		},
	})

	writer.Write(bytes)
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
func (manager *AdminManager)handleApi(writer http.ResponseWriter, _ *http.Request, path string) {
	api, ok := adminApiMapping[path]
	var response Map
	if !ok {
		response = Map {
			"code": 404,
			"message": "Not Found",
			"data": Api{},
		}
	} else {
		response = Map {
			"code": 200,
			"message": "Success",
			"data": api,
		}
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}

	fmt.Fprint(writer, string(bytes))
}

// /@api/path/year/:year/month/:month/day/:day
// 日统计
func (manager *AdminManager)handleApiDay(writer http.ResponseWriter, _ *http.Request, path string, year int, month int, day int) {
	apiStat := statManager.FindAvgStatForDay(path, year, month, day)
	minutes := statManager.FindMinuteStatForDay(path, year, month, day)

	bytes, err := json.Marshal(Map {
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

	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}

	writer.Write(bytes)
}

// /@api/[:path]/delete
// 删除API
func (manager *AdminManager)handleApiDelete(writer http.ResponseWriter, _ *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if !ok {
		_bytes, _ := json.Marshal(Map {
			"code": 404,
			"message": "Not found",
			"data": nil,
		})
		writer.Write(_bytes)
	} else {
		now := time.Now()
		timeString := fmt.Sprintf("%d%02d%02d_%02d%02d%02d", now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())
		newFile := api.File + "." + timeString + ".deleted"
		os.Rename(api.File, newFile)
		appManager.reload()

		_bytes, _ := json.Marshal(Map {
			"code": 200,
			"message": "Success",
			"data": newFile,
		})
		writer.Write(_bytes)
	}
}

// /@api/[:path]/update
// 更改API信息
func (manager *AdminManager)handleApiUpdate(writer http.ResponseWriter, request *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if !ok {
		_bytes, _ := json.Marshal(Map {
			"code": 404,
			"message": "Not found",
			"data": nil,
		})
		writer.Write(_bytes)
	} else {
		newBytes, err := ioutil.ReadAll(request.Body)
		if err != nil {
			_bytes, _ := json.Marshal(Map {
				"code": 500,
				"message": err.Error(),
				"data": nil,
			})
			writer.Write(_bytes)
			return
		}
		log.Println(api.Path, string(newBytes))

		ioutil.WriteFile(api.File, newBytes, 0777)

		appManager.reload()

		_bytes, _ := json.Marshal(Map {
			"code": 200,
			"message": "Success",
			"data": nil,
		})
		writer.Write(_bytes)
	}
}

// /@api/[:path]/rename
// 更改API文件名称
func (manager *AdminManager)handleApiRename(writer http.ResponseWriter, _ *http.Request, path string, toFile string) {
	api, ok := adminApiMapping[path]
	if !ok {
		_bytes, _ := json.Marshal(Map {
			"code": 404,
			"message": "Not found",
			"data": nil,
		})
		writer.Write(_bytes)
	} else {
		newFile := appManager.AppDir + string(os.PathSeparator) + "apis" + string(os.PathSeparator) + toFile
		os.Rename(api.File, newFile)
		appManager.reload()

		_bytes, _ := json.Marshal(Map {
			"code": 200,
			"message": "Success",
			"data": newFile,
		})
		writer.Write(_bytes)
	}
}

// /@api/[:path]/debug/logs
// 打印调试日志
func (manager *AdminManager)handleDebugLogs(writer http.ResponseWriter, _ *http.Request, path string) {
	logs := statManager.FindDebugLogsForPath(path)
	bytes, err := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": len(logs),
			"logs": logs,
		},
	})

	if err != nil {
		fmt.Println(writer, err.Error())
		return
	}
	writer.Write(bytes)
}

// /@api/[:path]/debug/flush
// 刷新调试日志
func (manager *AdminManager)handleDebugFlush(writer http.ResponseWriter, _ *http.Request, _ string) {
	err, count := statManager.FlushDebugLogs()
	if err != nil {
		bytes, _ := json.Marshal(Map {
			"code": 500,
			"message": err.Error(),
			"data": Map {
				"count": count,
			},
		})
		writer.Write(bytes)
		return
	}

	bytes, _ := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
	writer.Write(bytes)
}

// /@api/all
// 输出所有API信息
func (manager *AdminManager)handleApis(writer http.ResponseWriter, _ *http.Request) {
	//统计相关
	var arr = ApiArray
	for index, api := range arr {
		api.Stat = statManager.AvgStat(api.Path)
		arr[index] = api
	}

	response := Map {
		"code": 200,
		"message": "Success",
		"data": ApiArray,
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}
	fmt.Fprint(writer, string(bytes))
}

// /@api/reload
// 刷新API配置
func (manager *AdminManager)handleReloadApis(writer http.ResponseWriter) {
	appManager.reload()

	writer.Write([]byte(`{
	"code": 200,
	"message": "Success",
	"data": null
}`))
}

// /@cache/clear
// 清除所有缓存
func (manager *AdminManager)handleCacheClear(writer http.ResponseWriter) {
	count := cacheManager.ClearAll()

	bytes, _ := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
	writer.Write(bytes)
}

// /@cache/[:path]/clear
// 清除某个API对应的所有Cache
func (manager *AdminManager)handleCacheClearPath(writer http.ResponseWriter, path string)  {
	count := cacheManager.DeleteTag("$MeloyAPI$" + path)

	bytes, _ := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
	writer.Write(bytes)
}

// /@cache/tag/:tag/delete
// 删除某个标签对应的缓存
func (manager *AdminManager)handleCacheDeleteTag(writer http.ResponseWriter, tag string) {
	count := cacheManager.DeleteTag(tag)

	bytes, _ := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"count": count,
		},
	})
	writer.Write(bytes)
}

// /@cache/tag/:tag
// 打印某个标签信息
func (manager *AdminManager)handleCacheTagInfo(writer http.ResponseWriter, tag string) {
	count, keys, ok := cacheManager.StatTag(tag)
	if !ok {
		bytes, _ := json.Marshal(Map {
			"code": 404,
			"message": "Not found",
			"data": Map {
				"count": count,
				"keys": keys,
			},
		})

		writer.Write(bytes)
	} else {
		bytes, _ := json.Marshal(Map {
			"code": 200,
			"message": "Success",
			"data": Map {
				"count": count,
				"keys": keys,
			},
		})

		writer.Write(bytes)
	}
}

// /@git/pull
// 处理Git Pull命令
func (manager *AdminManager)handleGitPull(writer http.ResponseWriter) {
	cmd := exec.Command("sh", "-c", "cd " + appManager.AppDir + ";git pull;touch /tmp/tmp-go-file")

	stdout, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		manager.writeErrorMessage(writer, stdoutErr)
		return
	}

	runErr := cmd.Start()
	reader := bufio.NewReader(stdout)

	output := ""
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil || io.EOF == readErr {
			break
		}

		output += line
	}

	cmd.Wait()

	if runErr != nil {
		manager.writeErrorMessage(writer, runErr)
		return
	}

	//刷新数据
	go appManager.reload()

	_bytes, err := json.Marshal(Map {
		"code": 200,
		"message": output,
		"data": nil,
	})
	if err != nil {
		log.Println(err.Error())
	} else {
		writer.Write(_bytes)
	}
}

// /@monitor
// 监控信息
func (manager *AdminManager)handleMonitor(writer http.ResponseWriter) {
	//内存信息
	memoryStat := runtime.MemStats{}
	runtime.ReadMemStats(&memoryStat)

	//负载信息
	load1 := "0"
	load2 := "0"
	load3 := "0"
	func () {
		cmd := exec.Command("uptime")
		stdout, stdoutErr := cmd.StdoutPipe()
		if stdoutErr != nil {
			return
		}

		runErr := cmd.Start()
		if runErr != nil {
			return
		}

		bytes, err := ioutil.ReadAll(stdout)
		if err != nil {
			return
		}

		resultString := string(bytes)
		reg, _ := regexp.Compile("load averages:\\s*(\\S+)\\s*(\\S+)\\s*(\\S+)")
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

	resultBytes, _ := json.Marshal(Map {
		"code": 200,
		"message": "Success",
		"data": Map {
			"memory": memoryStat.Sys,
			"routines": runtime.NumGoroutine(),
			"load1m": load1,
			"load5m": load2,
			"load15m": load3,
		},
	})
	writer.Write(resultBytes)
}

// 校验请求
func (manager *AdminManager)validateRequest(writer http.ResponseWriter, request *http.Request) bool {
	//取得IP
	reg, _ := regexp.Compile(":\\d+$")
	ip := reg.ReplaceAllString(request.RemoteAddr, "")
	if adminConfig.Allow.Clients != nil && len(adminConfig.Allow.Clients) > 0 {
		if !containsString(adminConfig.Allow.Clients, ip) {
			if ip != "[::1]" {
				_bytes, err := json.Marshal(Map {
					"code": 401,
					"message": "Forbidden",
					"data": nil,
				})

				if err != nil {
					manager.writeErrorMessage(writer, err)
				} else {
					writer.Write(_bytes)
				}

				return false
			}
		}
	}
	return true
}

// 写入错误信息
func (manager *AdminManager)writeErrorMessage(writer http.ResponseWriter, err error) {
	_bytes, err := json.Marshal(Map {
		"code": 500,
		"message": err.Error(),
		"data": nil,
	})
	if err != nil {
		log.Println(err.Error())
	} else {
		writer.Write(_bytes)
	}
}
