package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"text/template"
)

// IsURL checks if given string is a valid URL
func IsURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	return strings.HasPrefix(u.Scheme, "http")
}

type Binary struct {
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

func MapKeys(m interface{}) (keys []string) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return keys
	}

	for _, k := range v.MapKeys() {
		keys = append(keys, k.String())
	}
	return keys
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
	t, err := template.New("interpolate").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(&buf, variables)

	return buf.String(), err
}

func IsExitError(err error) bool {
	var e *exec.ExitError
	return errors.As(err, &e)
}
