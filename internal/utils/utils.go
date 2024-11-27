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
	"sync"
	"text/template"

	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/sirupsen/logrus"
)

// IsURL checks if given string is a valid URL
func IsURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	return strings.HasPrefix(u.Scheme, "http")
}

const REPLACE_CHAR_DEFAULT = " "

var (
	ErrInvalidOptionsEnvFile = errors.New("invalid options on envfile")
	ErrEnvfileFormatIncorrect = errors.New("envfile incorrect")
)

// Envile is a structure for storing the information required to generate an envfile which can be consumed
// by the specified binary
type Envfile struct {
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
	// PathValue points to the file to read in and compute using the modify/include/exclude instructions.
	PathValue   string `mapstructure:"path" yaml:"path,omitempty" json:"path,omitempty"`
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
	// mutex is not copieable - during denormalization we create a new instance
	// during generate the paths will be different
	// during read it is if path is provided
	mu sync.Mutex
	// generatedFilePath is the path to the generated file path, which holds the unique task name reference
	// It will be merged with env variables from os.Environ(), supplied `context.container.env`, contents of Path if not empty.
	// All Include/Exclude Modifications are applied to the final environment Key/Value pairs.
	//
	// Single file is injected via the --env-file option to the docker|podman command.
	generatedFilePath string
}

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
	// can be overridden by Opts
	e.GeneratedDir = ".taskctl"
	for _, o := range opts {
		o(e)
	}
	e.mu = sync.Mutex{}
	return e
}

func (e *Envfile) WithPath(path string) *Envfile {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.PathValue = path
	return e
}

func (e *Envfile) Path() string {
	return e.PathValue
}

func (e *Envfile) WithGeneratedPath(path string) *Envfile {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.generatedFilePath = path
	return e
}

func (e *Envfile) GeneratedPath() string {
	return e.generatedFilePath
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

// ReaderFromPath returns an io.ReaderCloser from provided path
// Returning false if the file does not exist or is unable to read it
func ReaderFromPath(envfile *Envfile) (io.ReadCloser, bool) {
	if envfile == nil {
		return nil, false
	}
	if fi, err := os.Stat(envfile.PathValue); fi != nil && err == nil {
		f, err := os.Open(envfile.PathValue)
		if err != nil {
			logrus.Debugf("unable to open %s", envfile.PathValue)
			return nil, false
		}
		return f, true
	}
	return nil, false
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

// TASKCTL_ENV_FILE is the default location of env file ingested by taskctl for every run.
const TASKCTL_ENV_FILE string = "taskctl.env"

// DefaultTaskctlEnv checks if there is a file in the current directory `taskctl.env`
// if we ingest it into the Env variable
// giving preference to the `taskctl.env` specified K/V.
//
// Or should this be done once on start up?
func DefaultTaskctlEnv() *variables.Variables {
	defaultVars := variables.NewVariables()
	if fi, err := os.Stat(TASKCTL_ENV_FILE); fi != nil && err == nil {
		f, err := os.Open(fi.Name()) // this will always be relative to the executable
		if err != nil {
			logrus.Debug("unable to open default taskctl.env")
			return defaultVars
		}

		m, err := ReadEnvFile(f)
		if err != nil {
			logrus.Debug("unable to read default taskctl.env")
			return defaultVars
		}
		return defaultVars.Merge(variables.FromMap(m))
	}
	// file does not exist return empty vars
	return defaultVars
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
