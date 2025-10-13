package ngen

import (
	"bytes"
	"os"
	"strings"
)

// 首字母大写驼峰
func stringToUpperCamelCase(str string) string {
	if len(str) == 0 {
		return ""
	}
	strArr := strings.Split(str, "_")
	var sb bytes.Buffer
	for _, s := range strArr {
		if len(s) == 0 {
			continue
		}
		sb.WriteString(strings.ToUpper(s[0:1]) + s[1:])
	}
	return sb.String()
}

// 首字母小写驼峰
func stringToLowerCamelCase(str string) string {
	if len(str) == 0 {
		return ""
	}
	strArr := strings.Split(str, "_")
	var sb bytes.Buffer
	for n, s := range strArr {
		if len(s) == 0 {
			continue
		}
		if n == 0 {
			sb.WriteString(s)
		} else {
			sb.WriteString(strings.ToUpper(s[0:1]) + s[1:])
		}
	}
	return sb.String()
}
func fileIsExist(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err == nil {
		return true
	}
	if os.IsExist(err) {
		return true
	}
	return false
}

// 只能创建目录，不能创建文件
func createDir(path string) error {
	if fileIsExist(path) {
		return nil
	}
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
