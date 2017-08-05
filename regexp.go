package MeloyApi

import "regexp"

var reuseRegexpMap = map[string]*regexp.Regexp{}

// 生成可重用的正则
func ReuseRegexpCompile(pattern string) (*regexp.Regexp, error) {
	reg, ok := reuseRegexpMap[pattern]
	if ok {
		return reg, nil
	}

	reg, err := regexp.Compile(pattern)
	if err == nil {
		reuseRegexpMap[pattern] = reg
	}
	return reg, err
}
