package util

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
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

func ReadStringsSlice(v interface{}) (arr []string) {
	if v == nil {
		return arr
	}

	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice:
		val := reflect.ValueOf(v)
		arr = make([]string, val.Len())
		for i := 0; i < val.Len(); i++ {
			vi := val.Index(i).Interface()
			if vi != nil {
				arr[i] = vi.(string)
			}
		}
	case reflect.String:
		arr = []string{reflect.ValueOf(v).String()}
	}

	return arr
}

func ReadStringsMap(v interface{}) (m map[string]string) {
	m = make(map[string]string)

	if v == nil {
		return
	}

	m = make(map[string]string)
	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice:
		val := reflect.ValueOf(v)
		m = make(map[string]string, val.Len())
		for i := 0; i < val.Len(); i++ {
			vi := val.Index(i).Interface()
			if vi != nil {
				vis, ok := vi.(string)
				if !ok {
					return
				}

				kv := strings.Split(vis, "=")
				m[kv[0]] = kv[1]
			}
		}
	case reflect.Map:
		iter := reflect.ValueOf(v).MapRange()
		for iter.Next() {
			k := iter.Key().Interface()
			v := iter.Value().Interface()

			var ok bool
			var ks, vs string
			if ks, ok = k.(string); !ok {
				return
			}

			if v != nil {
				if vs, ok = v.(string); !ok {
					return
				}
				m[ks] = vs
			}
		}
	}

	return
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

func Getcwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return cwd, err
	}

	return cwd, nil
}

func IsUrl(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	if u.Scheme != "" {
		return true
	}

	return false
}
