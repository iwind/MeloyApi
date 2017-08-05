package MeloyApi

import (
	"strings"
	"strconv"
	"errors"
)

// 判断slice中是否包含某个字符串
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// 从字符串中分析尺寸
func parseSizeFromString(sizeString string) (float64, error) {
	if len(sizeString) == 0 {
		return 0, nil
	}

	reg, _ := ReuseRegexpCompile("^([\\d.]+)\\s*(b|byte|bytes|k|m|g|kb|mb|gb)$")
	matches := reg.FindStringSubmatch(strings.ToLower(sizeString))
	if len(matches) == 3 {
		size, err := strconv.ParseFloat(matches[1], 32)
		if err != nil {
			return 0, err
		} else {
			unit := matches[2]
			if unit == "k" || unit == "kb" {
				size = size * 1024
			} else if unit == "m" || unit == "mb" {
				size = size * 1024 * 1024
			} else if unit == "g" || unit == "gb" {
				size = size * 1024 * 1024 * 1024
			}

			return size, nil
		}
	}
	return 0, errors.New("invalid string:" + sizeString)
}
