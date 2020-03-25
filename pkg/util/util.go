package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"text/template"
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

func LastLine(r io.Reader) (l string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l = scanner.Text()
	}

	return l
}

func RenderString(tmpl string, variables map[string]string) (string, error) {
	var buf bytes.Buffer
	t, err := template.New("test").Parse(tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(&buf, variables)

	return buf.String(), err
}
