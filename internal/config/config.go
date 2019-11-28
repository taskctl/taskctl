package config

import (
	"fmt"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"time"
)

const (
	ContextTypeLocal     = "local"
	ContextTypeContainer = "container"
	ContextTypeRemote    = "remote"

	ContextContainerProviderDocker        = "docker"
	ContextContainerProviderDockerCompose = "docker-compose"
	ContextContainerProviderKubectl       = "kubectl"
)

var loaded = make(map[string]bool)
var cfg *Config

func Get() *Config {
	return cfg
}

type Config struct {
	Import    []string
	Contexts  map[string]builder.ContextDefinition
	Pipelines map[string][]builder.StageDefinition
	Tasks     map[string]builder.TaskDefinition
	Watchers  map[string]builder.WatcherDefinition

	builder.WilsonConfigDefinition
}

func Load(file string) (*Config, error) {
	var err error
	cfg, err = LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		cfg = &Config{
			Contexts:  make(map[string]builder.ContextDefinition),
			Tasks:     make(map[string]builder.TaskDefinition),
			Pipelines: make(map[string][]builder.StageDefinition),
			Watchers:  make(map[string]builder.WatcherDefinition),
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
	localCfg, err := LoadFile(file)
	if err != nil {
		return nil, err
	}

	err = cfg.merge(localCfg)
	if err != nil {
		return nil, err
	}

	if _, ok := cfg.Contexts[ContextTypeLocal]; !ok {
		cfg.Contexts[ContextTypeLocal] = builder.ContextDefinition{Type: ContextTypeLocal}
	}

	log.Debugf("config %s loaded", file)
	return cfg, nil
}

func LoadGlobalConfig() (*Config, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	file := path.Join(h, ".wilson", "config.yaml")
	if !util.FileExists(file) {
		return nil, nil
	}

	return LoadFile(file)
}

func LoadFile(file string) (*Config, error) {
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

		lconfig, err := LoadFile(importFile)
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
		SSH           util.Executable `yaml:"ssh"`
		Debug         bool

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
			SSH struct {
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
			Command      interface{}
			Context      string
			Env          map[string]string
			Dir          string
			Timeout      *time.Duration
			AllowFailure bool `yaml:"allow_failure"`
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
		Contexts:  make(map[string]builder.ContextDefinition),
		Tasks:     make(map[string]builder.TaskDefinition),
		Pipelines: make(map[string][]builder.StageDefinition),
		Watchers:  make(map[string]builder.WatcherDefinition),
		Import:    container.Import,
		WilsonConfigDefinition: builder.WilsonConfigDefinition{
			Shell:         container.Shell,
			Docker:        container.Docker,
			DockerCompose: container.DockerCompose,
			Kubectl:       container.Kubectl,
			Ssh:           container.SSH,
			Debug:         container.Debug,
		},
	}

	for name, def := range container.Contexts {
		cfg.Contexts[name] = builder.ContextDefinition{
			Type: def.Type,
			Dir:  def.Dir,
			Container: builder.ContainerDefinition{
				Provider:   def.Container.Provider,
				Name:       def.Container.Name,
				Image:      def.Container.Image,
				Exec:       def.Container.Exec,
				Options:    def.Container.Options,
				Env:        def.Container.Env,
				Executable: util.Executable{},
			},
			SSH: builder.SSHConfigDefinition{
				Options:    def.SSH.Options,
				User:       def.SSH.User,
				Host:       def.SSH.Host,
				Executable: def.SSH.Executable,
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
		cfg.Tasks[name] = builder.TaskDefinition{
			Name:    name,
			Command: util.ReadStringsSlice(def.Command),
			Context: def.Context,
			Env:     def.Env,
			Dir:     def.Dir,
			Timeout: def.Timeout,
		}
	}

	for name, stages := range container.Pipelines {
		cfg.Pipelines[name] = make([]builder.StageDefinition, len(stages))
		for i, def := range stages {
			switch reflect.TypeOf(def).Kind() {
			case reflect.Map:
				stage, ok := def.(map[interface{}]interface{})
				if !ok {
					return fmt.Errorf("pipeline %s unmarshalling error", name)
				}

				for k, v := range stage {
					switch k.(string) {
					case "allow_failure":
						cfg.Pipelines[name][i].AllowFailure = v.(bool)
					case "task":
						cfg.Pipelines[name][i].Task = v.(string)
						if cfg.Pipelines[name][i].Name == "" {
							cfg.Pipelines[name][i].Name = v.(string)
						}
					case "pipeline":
						cfg.Pipelines[name][i].Pipeline = v.(string)
						if cfg.Pipelines[name][i].Name == "" {
							cfg.Pipelines[name][i].Name = v.(string)
						}
					case "depends_on":
						cfg.Pipelines[name][i].DependsOn = util.ReadStringsSlice(v)
					case "name":
						cfg.Pipelines[name][i].Name = v.(string)
					case "env":
						envs, ok := v.(map[interface{}]interface{})
						if !ok {
							return fmt.Errorf("failed to parse %s envs", name)
						}
						cfg.Pipelines[name][i].Env = make(map[string]string)
						for kk, vv := range envs {
							cfg.Pipelines[name][i].Env[kk.(string)] = vv.(string)
						}
					default:
						return fmt.Errorf("failed to parse pipeline %s, unknown key %s", k.(string))
					}
				}
			case reflect.String:
				task := reflect.ValueOf(def).String()
				cfg.Pipelines[name][i] = builder.StageDefinition{
					Name: task,
					Task: task,
					Env:  make(map[string]string),
				}
			}
		}
	}

	for name, def := range container.Watchers {
		cfg.Watchers[name] = builder.WatcherDefinition{
			Events: util.ReadStringsSlice(def),
			Watch:  util.ReadStringsSlice(def),
			Task:   def.Task,
		}
	}

	*c = cfg

	return nil
}
