package MeloyApi

import (
	"net/http"
)

// 转发请求之前调用
func beforeHook(writer http.ResponseWriter, request *http.Request, api Api, do func ()) {
	do()
}

// 转发请求之后调用
func afterHook(writer http.ResponseWriter, request *http.Request, response *http.Response, api Api, err error) {

}