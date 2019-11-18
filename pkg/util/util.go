package util

import (
	"fmt"
	"log"
	"os"
	"reflect"
)

type Executable struct {
	Bin  string
	Args []string
}

func ConvertEnv(env map[string]string) []string {
	var i int
	enva := make([]string, len(env))
	for k, v := range env {
		enva[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}

	return enva
}

func FileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func ReadStringsArray(v interface{}) (arr []string) {
	if v == nil {
		return arr
	}

	iarr, ok := v.([]string)
	if ok {
		arr = make([]string, len(iarr))
		for i, el := range iarr {
			arr[i] = el
		}

		return arr
	}

	item, ok := v.(string)
	if ok {
		arr = []string{item}
	}

	return arr
}

func ListNames(m interface{}) (list []string) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return list
	}

	for _, k := range v.MapKeys() {
		list = append(list, k.String())
	}
	return list
}

func InArray(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}

	return false
}

func Getcwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	return cwd
}
