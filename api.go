package MeloyApi

import (
	"time"
	"regexp"
	"log"
	"encoding/json"
	"encoding/base64"
	"strings"
	"reflect"
)

// API定义
type Api struct {
	IsEnabled bool `json:"isEnabled"`

	Pattern string `json:"pattern"`
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
	MaxSize string `json:"maxSize"`

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
	countAddresses int

	patternRegexp regexp.Regexp
	patternNames []string

	responseString string
	hasResponseString bool
	timeoutDuration time.Duration
	maxSizeBits float64
}

// 分析API
func (api *Api) parse() {
	// 支持pattern，比如:name，:age，:subject(^[\\w-]+$)
	if len(api.Pattern) > 0 {
		if len(api.Path) == 0 {
			api.Path = api.Pattern
		}

		reg, err := regexp.Compile(":(?:(\\w+)(\\s*(\\([^)]+\\))?))")
		if err != nil {
			log.Fatal(err)
			return
		}

		pattern := api.Pattern
		matches := reg.FindAllStringSubmatch(pattern, 10)
		names := []string {}
		for _, match := range matches {
			names = append(names, match[1])
			if len(match[3]) > 0 {
				pattern = strings.Replace(pattern, match[0], match[3], 10)
			} else {
				pattern = strings.Replace(pattern, match[0], "(\\w+)", 10)
			}
		}

		reg, err = regexp.Compile(pattern)
		if err != nil {
			log.Println("Error:" + err.Error())
		} else {
			api.patternRegexp = *reg
			api.patternNames = names
		}
	}

	// 校验和转换api.methods
	for methodIndex, method := range api.Methods {
		api.Methods[methodIndex] = strings.ToUpper(method)
	}

	// 超时时间
	// 支持 100ms, 100s
	// 具体见：time.ParseDuration() 方法说明
	// such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
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

	// 最大请求尺寸
	size, err := parseSizeFromString(api.MaxSize)
	if err != nil {
		log.Println("Parse " + api.MaxSize + " Error:", err.Error())
	} else {
		api.maxSizeBits = size
	}

	// 响应数据
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

	//地址信息
	api.countAddresses = len(api.Addresses)
}

// 拷贝数据到另外一个API
func (api *Api) copyFrom(from Api) {
	value := reflect.ValueOf(api)
	value2 := reflect.ValueOf(from)

	fieldType := reflect.TypeOf(*api)

	countFields := value.Elem().NumField()

	for i := 0; i < countFields; i ++ {
		fieldValue := value2.Field(i)
		fieldName := fieldType.Field(i).Name

		if matched, _ := regexp.MatchString("^[a-z]", fieldName); matched {
			continue
		}

		value.Elem().Field(i).Set(fieldValue)
	}

	api.countAddresses = from.countAddresses

	api.patternRegexp = from.patternRegexp
	api.patternNames = from.patternNames

	api.responseString = from.responseString
	api.hasResponseString = from.hasResponseString
	api.timeoutDuration = from.timeoutDuration
	api.maxSizeBits = from.maxSizeBits
}