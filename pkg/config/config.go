package config

import (
	"errors"
	"fmt"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"time"
)

const (
	// todo: local, container, remote + providers
	CONTEXT_TYPE_LOCAL     = "local"
	CONTEXT_TYPE_CONTAINER = "container"
	CONTEXT_TYPE_REMOTE    = "remote"

	CONTEXT_CONTAINER_PROVIDER_DOCKER         = "docker"
	CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE = "docker-compose"
	CONTEXT_CONTAINER_PROVIDER_KUBECTL        = "kubectl"
)

var loaded = make(map[string]bool)
var cfg *Config

type WilsonConfig struct {
	Shell         util.Executable
	Docker        util.Executable
	DockerCompose util.Executable
	Kubectl       util.Executable
	Ssh           util.Executable
}

type ContextConfig struct {
	Type      string
	Dir       string
	Container Container
	Ssh       SshConfig
	Env       map[string]string
	Up        []string
	Down      []string
	Before    []string
	After     []string
	util.Executable
}

type Stage struct {
	Name      string
	Task      string
	DependsOn []string
	Env       map[string]string
}

type TaskConfig struct {
	Command []string
	Context string
	Env     map[string]string
	Dir     string
	Timeout *time.Duration
}

type WatcherConfig struct {
	Events []string
	Watch  []string
	Task   string
}

type Config struct {
	Import    []string
	Contexts  map[string]ContextConfig
	Pipelines map[string][]Stage
	Tasks     map[string]TaskConfig
	Watchers  map[string]WatcherConfig

	WilsonConfig
}

type Container struct {
	Provider string
	Name     string
	Image    string
	Exec     bool
	Options  []string
	Env      map[string]string
	util.Executable
}

type SshConfig struct {
	Options []string
	User    string
	Host    string
	util.Executable
}

func Get() *Config {
	return cfg
}

func Load(file string) (*Config, error) {
	var err error
	cfg, err = loadGlobalConfig()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		cfg = &Config{
			Contexts:  make(map[string]ContextConfig),
			Tasks:     make(map[string]TaskConfig),
			Pipelines: make(map[string][]Stage),
			Watchers:  make(map[string]WatcherConfig),
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if file == "" {
		file = "wilson.yaml"
		if !util.FileExists(path.Join(dir, file)) {
			return cfg, nil
		}
	}

	file = path.Join(dir, file)
	localCfg, err := load(file)
	if err != nil {
		return nil, err
	}

	err = cfg.merge(localCfg)
	if err != nil {
		return nil, err
	}

	if _, ok := cfg.Contexts[CONTEXT_TYPE_LOCAL]; !ok {
		cfg.Contexts[CONTEXT_TYPE_LOCAL] = ContextConfig{Type: CONTEXT_TYPE_LOCAL}
	}

	log.Debugf("config %s loaded", file)
	return cfg, nil
}

func loadGlobalConfig() (*Config, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	file := path.Join(h, ".wilson", "config.yaml")
	if !util.FileExists(file) {
		return nil, nil
	}

	return load(file)
}

func load(file string) (*Config, error) {
	loaded[file] = true
	config, err := readFile(file)
	if err != nil {
		return nil, err
	}

	importDir := path.Dir(file)
	for _, v := range config.Import {
		importFile := path.Join(importDir, v)
		if loaded[importFile] == true {
			continue
		}

		lconfig, err := load(importFile)
		if err != nil {
			return nil, err
		}
		err = lconfig.merge(config)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", importFile, err)
		}
		config = lconfig
	}

	return config, nil
}

func readFile(filename string) (*Config, error) {
	c := &Config{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}

	err = yaml.UnmarshalStrict(data, c)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}

	return c, nil
}

func (c *Config) merge(src *Config) error {
	if err := mergo.Merge(c, src); err != nil {
		return err
	}

	return nil
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var container struct {
		Shell         util.Executable `yaml:"shell"`
		Docker        util.Executable `yaml:"docker"`
		DockerCompose util.Executable `yaml:"docker-compose"`
		Kubectl       util.Executable `yaml:"kubectl"`
		Ssh           util.Executable `yaml:"ssh"`

		Import   []string
		Contexts map[string]struct {
			Type      string
			Dir       string
			Container struct {
				Provider string
				Name     string
				Image    string
				Exec     bool
				Options  []string
				Env      map[string]string
				util.Executable
			}
			Ssh struct {
				Options []string
				User    string
				Host    string
				util.Executable
			}
			Env    map[string]string
			Up     interface{}
			Down   interface{}
			Before interface{}
			After  interface{}
			util.Executable
		}
		Pipelines map[string][]interface{}
		Tasks     map[string]struct {
			Command interface{}
			Context string
			Env     map[string]string
			Dir     string
			Timeout *time.Duration
		}
		Watchers map[string]struct {
			Events []interface{}
			Watch  []interface{}
			Task   string
		}
	}
	if err := unmarshal(&container); err != nil {
		return err
	}

	cfg := Config{
		Contexts:  make(map[string]ContextConfig),
		Tasks:     make(map[string]TaskConfig),
		Pipelines: make(map[string][]Stage),
		Watchers:  make(map[string]WatcherConfig),
		Import:    container.Import,
		WilsonConfig: WilsonConfig{
			Shell:         container.Shell,
			Docker:        container.Docker,
			DockerCompose: container.DockerCompose,
			Kubectl:       container.Kubectl,
			Ssh:           container.Ssh,
		},
	}

	for name, def := range container.Contexts {
		cfg.Contexts[name] = ContextConfig{
			Type: def.Type,
			Dir:  def.Dir,
			Container: Container{
				Provider:   def.Container.Provider,
				Name:       def.Container.Name,
				Image:      def.Container.Image,
				Exec:       def.Container.Exec,
				Options:    def.Container.Options,
				Env:        def.Container.Env,
				Executable: util.Executable{},
			},
			Ssh: SshConfig{
				Options:    def.Ssh.Options,
				User:       def.Ssh.User,
				Host:       def.Ssh.Host,
				Executable: def.Ssh.Executable,
			},
			Env:        def.Env,
			Up:         util.ReadStringsSlice(def.Up),
			Down:       util.ReadStringsSlice(def.Down),
			Before:     util.ReadStringsSlice(def.Before),
			After:      util.ReadStringsSlice(def.After),
			Executable: def.Executable,
		}
	}

	for name, def := range container.Tasks {
		cfg.Tasks[name] = TaskConfig{
			Command: util.ReadStringsSlice(def.Command),
			Context: def.Context,
			Env:     def.Env,
			Dir:     def.Dir,
			Timeout: def.Timeout,
		}
	}

	for name, stages := range container.Pipelines {
		cfg.Pipelines[name] = make([]Stage, len(stages))
		for i, def := range stages {
			switch reflect.TypeOf(def).Kind() {
			case reflect.Map:
				stage, ok := def.(map[interface{}]interface{})
				if !ok {
					return errors.New("pipelines unmarshalling error")
				}

				for k, v := range stage {
					switch k.(string) {
					case "task":
						cfg.Pipelines[name][i].Task = v.(string)
					case "depends_on":
						cfg.Pipelines[name][i].DependsOn = util.ReadStringsSlice(v)
					case "name":
						cfg.Pipelines[name][i].Name = v.(string)
					}

					if cfg.Pipelines[name][i].Name == "" {
						cfg.Pipelines[name][i].Name = cfg.Pipelines[name][i].Task
					}
				}
			case reflect.String:
				task := reflect.ValueOf(def).String()
				cfg.Pipelines[name][i] = Stage{
					Name: task,
					Task: task,
					Env:  make(map[string]string),
				}
			}
		}
	}

	for name, def := range container.Watchers {
		cfg.Watchers[name] = WatcherConfig{
			Events: util.ReadStringsSlice(def),
			Watch:  util.ReadStringsSlice(def),
			Task:   def.Task,
		}
	}

	*c = cfg

	return nil
}
