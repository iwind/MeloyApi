package MeloyApi

import (
	"io"
	"net/http"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"regexp"
	"time"
	"strings"
	"math/rand"
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
	Path string
	Address string
	Methods []string

	Name string
	Description string
	Params []ApiParam

	AvailableAddresses []string
	File string
	Data string
}

type ApiParam struct {
	Name string
	Type string
	Description string
}

var ApiArray []Api

func LoadApp(configDir string) {
	servers := loadServers(configDir)

	appBytes, appErr := ioutil.ReadFile(configDir + "/app.json")
	if appErr != nil {
		fmt.Printf("Error:%s\n", appErr)
		return
	}

	var app App
	jsonError := json.Unmarshal(appBytes, &app)
	if jsonError != nil {
		fmt.Printf("Error:%s", jsonError)
		return
	}

	ApiArray = loadApis(configDir, servers)

	address := fmt.Sprintf("%s:%d", app.Host, app.Port)
	fmt.Printf("start %s:%d\n", app.Host, app.Port)

	//启动Server
	s := &http.Server{
		Addr: address,
		Handler: nil,
	}

	go (func (apis []Api) {
		for _, api := range apis {
			fmt.Println("load api '" + api.Path + "' from '" + api.File + "'")
			(func (api Api) {
				http.HandleFunc(api.Path, func (writer http.ResponseWriter, request *http.Request) {
					handle(writer, request, api)
				})
			})(api)
		}

		s.ListenAndServe()
	})(ApiArray)
}

func Wait()  {
	//Hold住进程
	for {
		time.Sleep(1 * time.Hour)
	}
}

func loadServers(configDir string) (servers []Server) {
	serverBytes, serverErr := ioutil.ReadFile(configDir + "/servers.json")

	if serverErr != nil {
		fmt.Printf("Error:%s\n", serverErr)
		return
	}

	jsonErr := json.Unmarshal(serverBytes, &servers)
	if jsonErr != nil {
		fmt.Printf("Error:%s\n", jsonErr)
		return
	}
	return
}

func loadApis(configDir string, servers []Server) (apis []Api) {
	files, err := ioutil.ReadDir(configDir + "/apis")
	if err != nil {
		fmt.Printf("Error:%s\n", err)
		return
	}

	reg, err := regexp.Compile("\\.json$")
	if err != nil {
		fmt.Printf("Error:%s\n", err)
		return
	}

	dataReg, err := regexp.Compile("\\.data\\.json")
	if err != nil {
		fmt.Printf("Error:%s\n", err)
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

		bytes, err := ioutil.ReadFile(configDir + "/apis/" + file.Name())
		if err != nil {
			fmt.Printf("Error:%s:%s\n", file.Name(), err)
			continue
		}

		var api Api
		jsonError := json.Unmarshal(bytes, &api)
		if jsonError != nil {
			fmt.Printf("Error:%s:%s\n", file.Name(), jsonError)
			continue
		}
		api.File = file.Name()

		//@TODO 校验和转换api.methods

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
					api.AvailableAddresses = append(api.AvailableAddresses, address)
					continue
				}

				weight := int(host.Weight * 10 / totalWeight)
				for i := 0; i < weight; i ++ {
					api.AvailableAddresses = append(api.AvailableAddresses, address)
				}
			}
		}

		//假数据
		fileName := file.Name()
		reg, _ := regexp.Compile("\\.json")
		dataFileName := configDir + "/apis/" + reg.ReplaceAllString(fileName, ".data.json")

		fileExists, _ := FileExists(dataFileName)
		if fileExists {
			bytes, err := ioutil.ReadFile(dataFileName)
			if err != nil {
				fmt.Println("Error:" + err.Error())
			} else {
				api.Data = string(bytes)
			}
		}

		apis = append(apis, api)
	}

	return
}

func handle(writer http.ResponseWriter, request *http.Request, api Api) {
	len := len(api.AvailableAddresses)

	if len == 0 {
		fmt.Fprintln(writer, "Does not have avaiable address")
		return
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Int() % len

	//检查method
	method := strings.ToLower(request.Method)
	if !contains(api.Methods, method) {
		fmt.Fprintln(writer, "'" + request.Method + "' method is not supported")
		return
	}

	address := api.AvailableAddresses[index]

	if strings.Compare(method, "get") == 0 {
		handleGet(writer, request, api, address)

	} else if strings.Compare(method, "post") == 0 {
		handlePost(writer, request, api, address)
	}

	//fmt.Fprintln(writer, "address:" + address + " path:" + request.RequestURI + " query:" + request.URL.RawQuery)
}

func handleGet(writer http.ResponseWriter, request *http.Request, api Api, address string) {
	query := request.URL.RawQuery
	url := address
	if len(query) > 0 {
		url += "?" + query
	}
	client := &http.Client{}
	newRequest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return
	}

	newRequest.Header = request.Header
	newRequest.Header.Set("Meloy-Api", "1.0")
	resp, err := client.Do(newRequest)

	if err != nil {
		resp.Body.Close()
		return
	}

	_copyHeaders(writer, resp)

	io.Copy(writer, resp.Body)
	resp.Body.Close()
}

func handlePost(writer http.ResponseWriter, request *http.Request, api Api, address string) {
	query := request.URL.RawQuery
	url := address
	if len(query) > 0 {
		url += "?" + query
	}

	request.ParseMultipartForm(1024 * 1024 * 1024 * 8)

	client := &http.Client{}
	newRequest, err := http.NewRequest("POST", url, strings.NewReader(request.PostForm.Encode()))

	if err != nil {
		return
	}

	newRequest.Header = request.Header
	newRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newRequest.Header.Set("Meloy-Api", "1.0")
	resp, err := client.Do(newRequest)

	if err != nil {
		resp.Body.Close()
		return
	}

	_copyHeaders(writer, resp)

	io.Copy(writer, resp.Body)
	resp.Body.Close()
}

func _copyHeaders(writer http.ResponseWriter, resp *http.Response)  {
	for key, value := range resp.Header {
		if len(value) != 1 {
			continue
		}

		if contains([]string { "Connection", "Server" }, key) {
			continue
		}

		writer.Header().Set(key, value[0])
	}

	writer.Header().Set("Server", "MeloyApi")

}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

