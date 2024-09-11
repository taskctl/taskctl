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
	"path/filepath"
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
	// Bin is the name of the executable to run
	// it must exist on the path
	// If using a default mvdn.sh context then
	// ensure it is on your path as symlink if you are only using aliases.
	Bin  string   `mapstructure:"bin" yaml:"bin" json:"bin"`
	Args []string `mapstructure:"args" yaml:"args,omitempty" json:"args,omitempty"`
}

// Envile is a structure for storing the information required to generate an envfile which can be consumed
// by the specified binary
type Envfile struct {
	// Generate will toggle the creation of the envFile
	// this "envFile" is only used in executables of type `docker|podman`
	Generate bool `mapstructure:"generate" yaml:"generate,omitempty" json:"generate,omitempty"`
	// list of variables to be excluded
	// from the injection into container runtimes
	//
	// Currently this is based on a prefix
	//
	// Example:
	// HOME=foo,HOMELAB=bar
	//
	// Both of these will be skipped
	Exclude []string `mapstructure:"exclude" yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Include []string `mapstructure:"include" yaml:"include,omitempty" json:"include,omitempty"`
	// Path is generated using task name and current timestamp
	// TODO: include additional graph info about the execution
	// e.g. owning pipeline (if any) execution number
	Path        string `mapstructure:"path" yaml:"path,omitempty" json:"path,omitempty"`
	ReplaceChar string `mapstructure:"replace_char" yaml:"replace_char,omitempty" json:"replace_char,omitempty"`
	Quote       bool   `mapstructure:"quote" yaml:"quote,omitempty" json:"quote,omitempty"`
	// Delay can be removed
	Delay int `mapstructure:"delay" yaml:"delay,omitempty" json:"delay,omitempty"`
	// Modify specifies the modifications to make to each env var and whether it meets the criteria
	// example:
	// - pattern: "^(?P<keyword>TF_VAR_)(?P<varname>.*)"
	// 	 operation: lower
	// the inputs are validated at task/pipeline build time and will fail if the
	// <keyword> and <varname> sub expressions are not present in the `pattern`
	Modify []ModifyEnv `mapstructure:"modify" yaml:"modify,omitempty" json:"modify,omitempty"`
	// defaults to .taskctl in the current directory
	// again this should be hidden from the user...
	GeneratedDir string `mapstructure:"generated_dir" yaml:"generated_dir,omitempty" json:"generated_dir,omitempty"`
}

const REPLACE_CHAR_DEFAULT = " "

var ErrInvalidOptionsEnvFile = errors.New("invalid options on envfile")

// Validate checks input is correct
//
// This will be added to later
func (e *Envfile) Validate() error {
	// validate modify
	for _, v := range e.Modify {
		if !v.IsValid() {
			return fmt.Errorf("%s, %w", "modify pattern", ErrInvalidOptionsEnvFile)
		}
	}
	return nil
}

type ModifyEnv struct {
	Pattern   string `mapstructure:"pattern" yaml:"pattern" json:"pattern"`
	Operation string `mapstructure:"operation" yaml:"operation" json:"operation" jsonschema:"enum=upper,enum=lower"`
}

func (me ModifyEnv) IsValid() bool {
	return strings.Contains(me.Pattern, "keyword") && strings.Contains(me.Pattern, "varname")
}

// Opts is a task runner configuration function.
type EnvFileOpts func(*Envfile)

// NewEnvFile creates a new instance of the EnvFile
// initializes it with some defaults
func NewEnvFile(opts ...EnvFileOpts) *Envfile {
	e := &Envfile{}
	e.ReplaceChar = REPLACE_CHAR_DEFAULT
	// e.Path = "envfile"
	e.GeneratedDir = ".taskctl"
	for _, o := range opts {
		o(e)
	}
	return e
}

// ConvertEnv converts map representing the environment to array of strings in the form "key=value"
func ConvertEnv(env map[string]string) []string {
	i := 0
	enva := make([]string, len(env))
	for k, v := range env {
		enva[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}

	return enva
}

// ConvertFromEnv takes a string array and coverts it to a map of strings
// since an env variable can only really be a string
// it's safe to convert to string and not interface
// downstream programs need to cast values to what they expect
func ConvertFromEnv(env []string) map[string]string {
	envMap := make(map[string]string)
	for _, val := range env {
		v := strings.Split(val, "=")
		envMap[v[0]] = v[1]
	}
	return envMap
}

// ConvertToMapOfStrings converts map of interfaces to map of strings
func ConvertToMapOfStrings(m map[string]interface{}) map[string]string {
	mdst := make(map[string]string)

	for k, v := range m {
		mdst[k] = fmt.Sprintf("%v", v)
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

// GetFullPath
func GetFullPath(path string) string {
	fileIsLocal := filepath.IsLocal(path)
	if fileIsLocal {
		return filepath.Join(MustGetwd(), path)
	}
	return path
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

// ConvertStringToMachineFriendly takes astring and replaces
// any occurence of non machine friendly chars with machine friendly ones
func ConvertStringToMachineFriendly(str string) string {
	// These pairs can be extended cane
	return strings.NewReplacer(":", "_", ` `, "__").Replace(str)
}

// ConvertStringToHumanFriendly takes a ConvertStringToMachineFriendly generated string and
// and converts it back to its original human friendly form
func ConvertStringToHumanFriendly(str string) string {
	// Order is important
	// pass in the __ first to replace that with spaces
	// and only _ should be left to go back to :
	return strings.NewReplacer("__", ` `, "_", ":").Replace(str)
}
