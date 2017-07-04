package MeloyApi

import (
	"fmt"
	"net/http"
	"encoding/json"
)

func LoadUI(webDir string, host string, port int)  {
	address := fmt.Sprintf("%s:%d", host, port)
	fmt.Println("start " + address)
	s := &http.Server{
		Addr: address,
		Handler: nil,
	}

	go (func() {
		fs := http.FileServer(http.Dir(webDir))
		http.Handle("/", fs)

		http.HandleFunc("/apis", handleUIApis)

		s.ListenAndServe()
	})()
}

func handleUIApis(writer http.ResponseWriter, request *http.Request) {
	bytes, err := json.MarshalIndent(ApiArray, "", "    ")
	if err != nil {
		fmt.Fprint(writer, err.Error())
		return
	}
	fmt.Fprint(writer, string(bytes))
}
