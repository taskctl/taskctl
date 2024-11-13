package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
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
		// ensure vars with `=` are not truncated
		envMap[v[0]] = strings.Join(v[1:], "=")
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
		return escapeWinPaths(filepath.Join(MustGetwd(), path))
	}
	return escapeWinPaths(path)
}

func escapeWinPaths(path string) string {
	return strings.NewReplacer(`\`, `\\`).Replace(path)
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
func ReadEnvFile(r io.ReadCloser) (map[string]string, error) {
	envs := make(map[string]string)
	envscanner := bufio.NewScanner(r)
	defer r.Close()
	for envscanner.Scan() {
		kv := strings.Split(envscanner.Text(), "=")
		// ensure an unset variable gets passed
		// through as zerolength string
		if len(kv) >= 2 {
			// ensure EnvVar values which themselves include
			// an `=` equals are correctly set
			envs[kv[0]] = strings.Join(kv[1:], "=")
		}
	}

	if err := envscanner.Err(); err != nil {
		return nil, err
	}

	return envs, nil
}

// ConvertToMachineFriendly converts a string containing characters that would not play nice key names
// e.g. `->` or `|` `/` `\\` `:`
// To make it easier to decipher the names we replace these characters with a map to a known sequence.
//
//	replaceSequence := []string{
//			"->", "__a__",
//			`|`, "__b__",
//			`/`, "__c__",
//			`\`, "__d__",
//			`:`, "__e__",
//			` `, "__f__", // space sequence
//	}
func ConvertToMachineFriendly(str string) string {
	replaceSequence := []string{
		"->", "__a__",
		`|`, "__b__",
		`/`, "__c__",
		`\`, "__d__",
		`:`, "__e__",
		` `, "__f__",
	}
	return strings.NewReplacer(replaceSequence...).Replace(str)
}

// EncodeBase62 takes a string and converts it
// to base62 format - this is safer than using regex or strings replace.
func EncodeBase62(str string) string {
	return base62EncodeToString([]byte(str))
}

// DecodeBase62 takes a EncodeBase62 generated string and
// and converts it back to its original human friendly form
func DecodeBase62(str string) string {
	// Order is important
	// pass in the __ first to replace that with spaces
	// and only _ should be left to go back to :
	if decoded, err := base62DecodeString(str); err == nil {
		return string(decoded)
	}
	return ""
}

const PipelineDirectionChar string = "->"

// CascadeName builds the name using the ancestors with a pipeline separator
func CascadeName(parents []string, current string) string {
	return fmt.Sprintf("%s%s%s", strings.Join(parents, PipelineDirectionChar), PipelineDirectionChar, current)
}

// TailExtract takes the last possible node from a pipeline string
func TailExtract(v string) string {
	split := strings.Split(v, PipelineDirectionChar)
	return split[len(split)-1]
}

// base62 helpers included here - to avoid introducing a secondary dependancy
// NOTE: this is by far not the most performant method
// performance here is not an issue
func base62EncodeToString(v []byte) string {
	var i big.Int
	i.SetBytes(v[:])
	return i.Text(62)
}

func base62DecodeString(s string) ([]byte, error) {
	var i big.Int
	_, ok := i.SetString(s, 62)
	if !ok {
		return nil, fmt.Errorf("cannot parse base62: %q", s)
	}
	return i.Bytes(), nil
}
