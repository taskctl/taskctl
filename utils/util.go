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

// Binary is a structure for storing binary file path and arguments that should be passed on binary's invocation
type Binary struct {
	Bin  string
	Args []string
}

// ConvertEnv converts map representing the environment to array of strings in the form "key=value"
func ConvertEnv(env map[string]string) []string {
	var i int
	enva := make([]string, len(env))
	for k, v := range env {
		enva[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}

	return enva
}

// ConvertToMapOfStrings converts map of interfaces to map of strings
func ConvertToMapOfStrings(m map[string]interface{}) map[string]string {
	mdst := make(map[string]string)

	for k, v := range m {
		mdst[k] = v.(string)
	}
	return mdst
}

// FileExists checks if the file exists
func FileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

// MapKeys returns an array of map's keys
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

// LastLine returns last line from provided reader
func LastLine(r io.Reader) (l string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		l = scanner.Text()
	}

	return l
}

// RenderString parses given string as a template and executes it with provided params
func RenderString(tmpl string, variables map[string]interface{}) (string, error) {
	funcMap := template.FuncMap{
		"default": func(arg interface{}, value interface{}) interface{} {
			v := reflect.ValueOf(value)
			switch v.Kind() {
			case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
				if v.Len() == 0 {
					return arg
				}
			case reflect.Bool:
				if !v.Bool() {
					return arg
				}
			default:
				return value
			}

			return value
		},
	}

	var buf bytes.Buffer
	t, err := template.New("interpolate").Funcs(funcMap).Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(&buf, variables)

	return buf.String(), err
}

// IsExitError checks if given error is an instance of exec.ExitError
func IsExitError(err error) bool {
	var e *exec.ExitError
	return errors.As(err, &e)
}

// MustGetwd returns current working directory.
// Panics is os.Getwd() returns error
func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return wd
}

// MustGetUserHomeDir returns current working directory.
// Panics is os.UserHomeDir() returns error
func MustGetUserHomeDir() string {
	hd, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return hd
}

// ReadEnvFile reads env file inv `k=v` format
func ReadEnvFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	envs := make(map[string]string)
	envscanner := bufio.NewScanner(f)
	for envscanner.Scan() {
		kv := strings.Split(envscanner.Text(), "=")
		envs[kv[0]] = kv[1]
	}

	if err := envscanner.Err(); err != nil {
		return nil, err
	}

	return envs, nil
}
