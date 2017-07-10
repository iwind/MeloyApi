package MeloyApi

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"log"
	"fmt"
	"regexp"
	"strconv"
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

type AdminApiListResponse struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data []Api `json:"data"`
}

type AdminApiResponse struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data Api `json:"data"`
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

	//初始化应用
	adminApiMapping = make(map [string] Api)
	for _, api := range ApiArray {
		adminApiMapping[api.Path] = api //@TODO 支持  /:name/:age => /abc/123
	}

	go func() {
		serverMux := http.NewServeMux()
		serverMux.HandleFunc("/", manager.handleRequest)

		http.ListenAndServe(address, serverMux)
	}()
}

// 处理请求
func (manager *AdminManager)handleRequest(writer http.ResponseWriter, request *http.Request)  {
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

	{
		fmt.Fprint(writer, "404 page not found")
	}
}

// 处理API根目录请求
func (manager *AdminManager)handleIndex(writer http.ResponseWriter) {
	bytes, _ := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			Version string `json:"version"`
		} `json:"data"`
	} {
		200,
		"Success",
		struct{
			Version string `json:"version"`
		}{
			Version: MELOY_API_VERSION,
		},
	})
	writer.Write(bytes)
}

func (manager *AdminManager)handleMock(writer http.ResponseWriter, _ *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if ok && len(api.Mock) > 0 {
		fmt.Fprint(writer, api.Mock)
	}
}

func (manager *AdminManager)handleApi(writer http.ResponseWriter, _ *http.Request, path string) {
	api, ok := adminApiMapping[path]
	var response AdminApiResponse
	if !ok {
		response = AdminApiResponse {
			404,
			"Not Found",
			Api{},
		}
	} else {
		response = AdminApiResponse{
			200,
			"Success",
			api,
		}
	}

	bytes, err := json.Marshal(response)
	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}

	fmt.Fprint(writer, string(bytes))
}

func (manager *AdminManager)handleApiDay(writer http.ResponseWriter, _ *http.Request, path string, year int, month int, day int) {
	apiStat := statManager.FindAvgStatForDay(path, year, month, day)
	minutes := statManager.FindMinuteStatForDay(path, year, month, day)

	bytes, err := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			AvgMs int `json:"avgMs"`
			Requests int `json:"requests"`
			Hits int `json:"hits"`
			Errors int `json:"errors"`
			Minutes []ApiMinuteStat `json:"minutes"`
		} `json:"data"`
	}{
		Code: 200,
		Message: "Success",
		Data: struct{
			AvgMs int `json:"avgMs"`
			Requests int `json:"requests"`
			Hits int `json:"hits"`
			Errors int `json:"errors"`
			Minutes []ApiMinuteStat `json:"minutes"`
		} {
			AvgMs:apiStat.AvgMs,
			Requests:apiStat.Requests,
			Hits: apiStat.Hits,
			Errors: apiStat.Errors,
			Minutes:minutes,
		},
	})

	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}

	writer.Write(bytes)
}

func (manager *AdminManager)handleDebugLogs(writer http.ResponseWriter, _ *http.Request, path string) {
	logs := statManager.FindDebugLogsForPath(path)
	bytes, err := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			Count int `json:"count"`
			Logs []DebugLog `json:"logs"`
		} `json:"data"`
	}{
		200,
		"Success",
		struct{
			Count int `json:"count"`
			Logs []DebugLog `json:"logs"`
		} {
			len(logs),
			logs,
		},
	})

	if err != nil {
		fmt.Println(writer, err.Error())
		return
	}
	writer.Write(bytes)
}

func (manager *AdminManager)handleDebugFlush(writer http.ResponseWriter, _ *http.Request, _ string) {
	err, count := statManager.FlushDebugLogs()
	if err != nil {
		bytes, _ := json.Marshal(struct {
			Code    int `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Count int `json:"count"`
			} `json:"data"`
		}{
			500,
			err.Error(),
			struct {
				Count int `json:"count"`
			} {
				count,
			},
		})
		writer.Write(bytes)
		return
	}

	bytes, _ := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Count int `json:"count"`
		} `json:"data"`
	}{
		200,
		"Success",
		struct {
			Count int `json:"count"`
		} {
			count,
		},
	})
	writer.Write(bytes)
}

func (manager *AdminManager)handleApis(writer http.ResponseWriter, _ *http.Request) {
	//统计相关
	var arr = ApiArray
	for index, api := range arr {
		api.Stat = statManager.AvgStat(api.Path)
		arr[index] = api
	}

	response := AdminApiListResponse{}
	response.Data = ApiArray
	response.Code = 200

	bytes, err := json.Marshal(response)
	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}
	fmt.Fprint(writer, string(bytes))
}

// 刷新API配置
func (manager *AdminManager)handleReloadApis(writer http.ResponseWriter) {
	appManager.reload()

	writer.Write([]byte(`{
	"code": 200,
	"message": "Success",
	"data": null
}`))
}

func (manager *AdminManager)handleCacheClear(writer http.ResponseWriter) {
	count := cacheManager.ClearAll()

	bytes, _ := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			Count int `json:"count"`
		} `json:"data"`
	} {
		200,
		"Welcome to MeloyAPI",
		struct{
			Count int `json:"count"`
		}{
			Count: count,
		},
	})
	writer.Write(bytes)
}

// 清除某个API对应的所有Cache
func (manager *AdminManager)handleCacheClearPath(writer http.ResponseWriter, path string)  {
	count := cacheManager.DeleteTag("$MeloyAPI$" + path)

	bytes, _ := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			Count int `json:"count"`
		} `json:"data"`
	} {
		200,
		"Success",
		struct{
			Count int `json:"count"`
		}{
			Count:count,
		},
	})
	writer.Write(bytes)
}

// 删除某个标签对应的缓存
func (manager *AdminManager)handleCacheDeleteTag(writer http.ResponseWriter, tag string) {
	count := cacheManager.DeleteTag(tag)

	bytes, _ := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			Count int `json:"count"`
		} `json:"data"`
	} {
		200,
		"Success",
		struct{
			Count int `json:"count"`
		}{
			Count:count,
		},
	})
	writer.Write(bytes)
}

// 删除某个标签信息
func (manager *AdminManager)handleCacheTagInfo(writer http.ResponseWriter, tag string) {
	count, keys, ok := cacheManager.StatTag(tag)
	if !ok {
		bytes, _ := json.Marshal(struct {
			Code int `json:"code"`
			Message string `json:"message"`
			Data struct{
				Count int `json:"count"`
				Keys []string `json:"keys"`
			} `json:"data"`
		} {
			404,
			"Not found",
			struct{
				Count int `json:"count"`
				Keys []string `json:"keys"`
			}{
				Count:count,
				Keys:keys,
			},
		})

		writer.Write(bytes)
	} else {
		bytes, _ := json.Marshal(struct {
			Code int `json:"code"`
			Message string `json:"message"`
			Data struct{
				Count int `json:"count"`
				Keys []string `json:"keys"`
			} `json:"data"`
		} {
			200,
			"Success",
			struct{
				Count int `json:"count"`
				Keys []string `json:"keys"`
			}{
				Count:count,
				Keys:keys,
			},
		})

		writer.Write(bytes)
	}
}

// 校验请求
func (manager *AdminManager)validateRequest(writer http.ResponseWriter, request *http.Request) bool {
	//取得IP
	reg, _ := regexp.Compile(":\\d+$")
	ip := reg.ReplaceAllString(request.RemoteAddr, "")
	if adminConfig.Allow.Clients != nil && len(adminConfig.Allow.Clients) > 0 {
		if !containsString(adminConfig.Allow.Clients, ip) {
			if ip != "[::1]" {
				fmt.Fprint(writer, `{
					"code": "401",
					"message": "Forbidden",
					"data": null
				}`)
				return false
			}
		}
	}
	return true
}
