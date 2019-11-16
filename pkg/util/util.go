package util

import (
	"fmt"
	"os"
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

	iarr, ok := v.([]interface{})
	if ok {
		arr = make([]string, len(iarr))
		for i, dep := range iarr {
			arr[i] = dep.(string)
		}

		return arr
	}

	item, ok := v.(string)
	if ok {
		arr = []string{item}
	}

	return arr
}
