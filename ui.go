package MeloyApi

import (
	"fmt"
	"net/http"
	"encoding/json"
	"io/ioutil"
)

type AdminConfig struct {
	Host string
	Port int
}

type ApiListResponse struct {
	Code int
	Message string
	Data []Api
}

func LoadUI(webDir string)  {
	bytes, err := ioutil.ReadFile(webDir + "/app.json")
	if err != nil {
		fmt.Println("Error:" + err.Error())
		return
	}

	var adminConfig AdminConfig
	err = json.Unmarshal(bytes, &adminConfig)
	if err != nil {
		fmt.Println("Error:" + err.Error())
		return
	}

	address := fmt.Sprintf("%s:%d", adminConfig.Host, adminConfig.Port)
	fmt.Println("start " + address)

	go (func() {
		serverMux := http.NewServeMux()
		fs := http.FileServer(http.Dir(webDir))

		serverMux.Handle("/@admin/", http.StripPrefix("/@", fs))
		serverMux.HandleFunc("/", handleData)
		serverMux.HandleFunc("/@admin/apis", handleUIApis)

		http.ListenAndServe(address, serverMux)
	})()
}

func handleData(writer http.ResponseWriter, request *http.Request) {
	path := request.URL.Path
	mapping := make(map [string] Api)
	for _, api := range ApiArray {
		mapping[api.Path] = api //@TODO 支持  /:name/:age => /abc/123
	}

	api, ok := mapping[path]
	if ok && len(api.Data) > 0 {
		fmt.Fprint(writer, api.Data)
	}
}

func handleUIApis(writer http.ResponseWriter, request *http.Request) {
	response := ApiListResponse{}
	response.Data = ApiArray
	response.Code = 200

	bytes, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}
	fmt.Fprint(writer, string(bytes))
}
