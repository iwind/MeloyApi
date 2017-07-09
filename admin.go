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
func LoadAdmin(appDir string)  {
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
		//fs := http.FileServer(http.Dir(webDir))
		//serverMux.Handle("/@admin/", http.StripPrefix("/@", fs))
		serverMux.HandleFunc("/", handleAdminRequest)

		http.ListenAndServe(address, serverMux)
	}()
}

// 处理请求
func handleAdminRequest(writer http.ResponseWriter, request *http.Request)  {
	if !validateAdminRequest(writer, request) {
		return
	}

	path := request.URL.Path
	if path == "/@api/all" {
		handleAdminApis(writer, request)
		return
	}

	{
		reg, _ := regexp.Compile("^/@mock(/.+)$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			handleMock(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			handleAdminApi(writer, request, matches[1])
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
			handleAdminApiDay(writer, request, matches[1], year, month, day)
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api/\\[(.+)]/debug/logs$")
		matches := reg.FindStringSubmatch(path)
		if len(matches) > 0 {
			handleAdminDebugLogs(writer, request, matches[1])
			return
		}
	}

	{
		reg, _ := regexp.Compile("^/@api$")
		if reg.MatchString(path) {
			handleAdminIndex(writer)

			return
		}
	}

	{
		fmt.Fprint(writer, "404 page not found")
	}
}

func handleAdminIndex(writer http.ResponseWriter) {
	bytes, _ := json.Marshal(struct {
		Code int
		Message string
		Data struct{}
	} {
		200,
		"OK",
		struct{}{},
	})
	writer.Write(bytes)
}

func handleMock(writer http.ResponseWriter, _ *http.Request, path string) {
	api, ok := adminApiMapping[path]
	if ok && len(api.Mock) > 0 {
		fmt.Fprint(writer, api.Mock)
	}
}

func handleAdminApi(writer http.ResponseWriter, _ *http.Request, path string) {
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

func handleAdminApiDay(writer http.ResponseWriter, _ *http.Request, path string, year int, month int, day int) {
	apiStat := statManager.FindAvgStatForDay(path, year, month, day)
	minutes := statManager.FindMinuteStatForDay(path, year, month, day)

	bytes, err := json.Marshal(struct {
		Code int `json:"code"`
		Message string `json:"message"`
		Data struct{
			AvgMs int `json:"avgMs"`
			Requests int `json:"requests"`
			Minutes []ApiMinuteStat `json:"minutes"`
		} `json:"data"`
	}{
		Code: 200,
		Message: "Success",
		Data: struct{
			AvgMs int `json:"avgMs"`
			Requests int `json:"requests"`
			Minutes []ApiMinuteStat `json:"minutes"`
		} {
			AvgMs:apiStat.AvgMs,
			Requests:apiStat.Requests,
			Minutes:minutes,
		},
	})

	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}

	writer.Write(bytes)
}

func handleAdminDebugLogs(writer http.ResponseWriter, _ *http.Request, path string) {
	logs := statManager.FindDebugLogsForDay(path)
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

func handleAdminApis(writer http.ResponseWriter, _ *http.Request) {
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

// 校验请求
func validateAdminRequest(writer http.ResponseWriter, request *http.Request) bool {
	//取得IP
	reg, _ := regexp.Compile(":\\d+$")
	ip := reg.ReplaceAllString(request.RemoteAddr, "")
	if adminConfig.Allow.Clients != nil && len(adminConfig.Allow.Clients) > 0 {
		if !contains(adminConfig.Allow.Clients, ip) {
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
