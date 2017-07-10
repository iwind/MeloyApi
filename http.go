package MeloyApi

import (
	"net/http"
)

type ApiHasHandleFunc interface { //this is just so it would work for gorilla and http.ServerMux
	HandleFunc(pattern string, handler func(w http.ResponseWriter, req *http.Request))
}

//API处理器
type ApiHandler struct {
	http.HandlerFunc
	Api *Api
	Enabled bool
}
type ApiHandlers map[string] *ApiHandler

// 处理HTTP请求
func (h ApiHandlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if handler, ok := h[path]; ok && handler.Enabled {
		handler.ServeHTTP(w, r)
	} else {
		http.Error(w, "404 page not found", http.StatusNotFound)
	}
}

// 设置处理函数
func (h ApiHandlers) HandleFunc(mux ApiHasHandleFunc, api *Api, handler http.HandlerFunc) {
	pattern := api.Path
	h[pattern] = &ApiHandler{handler, api,true}
	mux.HandleFunc(pattern, h.ServeHTTP)
}
