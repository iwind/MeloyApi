package MeloyApi

import (
	"time"
	"regexp"
	"log"
	"encoding/json"
	"encoding/base64"
	"strings"
)

// API定义
type Api struct {
	Path string `json:"path"`
	Address string `json:"address"`
	Methods []string `json:"methods"`

	IsAsynchronous bool
	Response struct{
		String string `json:"string"`
		Binary string `json:"binary"`
		XML string `json:"xml"`
		JSON interface{} `json:"json"`
	}

	Headers []struct {
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`

	Timeout string `json:"timeout"`

	Name string `json:"name"`
	Description string `json:"description"`
	Params []ApiParam `json:"params"`
	Dones []string `json:"dones"`
	Todos []string `json:"todos"`
	IsDeprecated bool `json:"isDeprecated"`
	Version string `json:"version"`

	Roles []string `json:"roles"`
	Author string `json:"author"`
	Company string `json:"company"`

	Addresses []ApiAddress `json:"availableAddresses"`
	File string `json:"file"`
	Mock string `json:"mock"`

	Stat ApiStat `json:"stat"`

	// 分析后的数据
	responseString string
	hasResponseString bool
	timeoutDuration time.Duration
}

// 分析API
func (api *Api) Parse() {
	//校验和转换api.methods
	for methodIndex, method := range api.Methods {
		api.Methods[methodIndex] = strings.ToUpper(method)
	}

	//超时时间
	//支持 100ms, 100s
	if len(api.Timeout) > 0 {
		reg, _ := regexp.Compile("^(\\d+(?:\\.\\d+)?)\\s*(ms|s)$")
		matches := reg.FindStringSubmatch(api.Timeout)
		if len(matches) == 3 {
			duration, err := time.ParseDuration(matches[1] + matches[2])
			if err != nil {
				log.Println("API timeout parse failed '" + api.Timeout + "'")
			}
			api.timeoutDuration = duration
		} else {
			log.Println("API timeout parse failed '" + api.Timeout + "'")
		}
	}

	//响应数据
	if len(api.Response.String) > 0 {
		api.responseString = api.Response.String
		api.hasResponseString = true
	} else if len(api.Response.XML) > 0 {
		api.responseString = api.Response.XML
		api.hasResponseString = true
	} else if len(api.Response.Binary) > 0 {
		_bytes, err := base64.StdEncoding.DecodeString(api.Response.Binary)

		if err != nil {
			api.hasResponseString = false
		} else {
			api.responseString = string(_bytes)
			api.hasResponseString = true
		}
	} else if api.Response.JSON != nil {
		_bytes, err := json.Marshal(api.Response.JSON)
		if err != nil {
			api.hasResponseString = false
		} else {
			api.responseString = string(_bytes)
			api.hasResponseString = true
		}
	}
}

// 拷贝数据到另外一个API
func (api *Api) copyFrom(from Api) {
	api.Path = from.Path
	api.Methods = from.Methods
	api.Address = from.Address

	api.Headers = from.Headers
	api.Timeout = from.Timeout

	api.Name = from.Name
	api.Description = from.Description
	api.Params = from.Params
	api.Dones = from.Dones
	api.Todos = from.Todos
	api.IsDeprecated = from.IsDeprecated
	api.Version = from.Version

	api.Roles = from.Roles
	api.Author = from.Author
	api.Company = from.Company

	api.IsAsynchronous = from.IsAsynchronous
	api.Response = from.Response

	api.Addresses = from.Addresses
	api.File = from.File
	api.Mock = from.Mock

	api.Stat = from.Stat

	api.timeoutDuration = from.timeoutDuration
	api.hasResponseString = from.hasResponseString
	api.responseString = from.responseString
}