package MeloyApi

import (
	"net/http"
)

// API Handler管理器
type HandlerManager struct {
	handlers map[string]*ApiHandler
}

// API处理器
type ApiHandler struct {
	http.HandlerFunc
	Api       *Api
	isEnabled bool
}

// 约束HandleFunc
type ApiHasHandleFunc interface {
	HandleFunc(pattern string, handler func(w http.ResponseWriter, req *http.Request))
}

// 初始化
func (manager *HandlerManager) init() {
	manager.handlers = map[string]*ApiHandler{}
}

// 设置处理函数
func (manager *HandlerManager) HandleFunc(serverMux ApiHasHandleFunc, api *Api, handler http.HandlerFunc) {
	pattern := api.Path
	manager.handlers[pattern] = &ApiHandler{
		handler,
		api,
		true,
	}
	serverMux.HandleFunc(pattern, manager.handle)
}

// 处理HTTP请求
func (manager *HandlerManager) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if handler, ok := manager.handlers[path]; ok && handler.isEnabled {
		handler.ServeHTTP(w, r)
	} else {
		http.Error(w, "404 page not found ("+path+")", http.StatusNotFound)
	}
}

// 禁用所有的处理函数
func (manager *HandlerManager) disableAll() {
	for _, handler := range manager.handlers {
		handler.isEnabled = false
	}
}

// 查找API对应的处理函数
func (manager *HandlerManager) find(path string) (*ApiHandler, bool) {
	handler, ok := manager.handlers[path]
	return handler, ok
}
