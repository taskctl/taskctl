package config

import (
	"fmt"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
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
	Shell         util.Executable `yaml:"shell"`
	Docker        util.Executable `yaml:"docker"`
	DockerCompose util.Executable `yaml:"docker-compose"`
	Kubectl       util.Executable `yaml:"kubectl"`
	Ssh           util.Executable `yaml:"ssh"`
}

type ContextConfig struct {
	Type      string
	Dir       string
	Container Container
	Ssh       SshConfig
	Env       map[string]string
	Up        interface{}
	Down      interface{}
	Before    interface{}
	After     interface{}
	util.Executable
}

type Stage struct {
	StageName string `yaml:"name"`
	Task      string
	DependsOn interface{} `yaml:"depends_on"`
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
			Pipelines: make(map[string][]Stage, 0),
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

func Get() *Config {
	return cfg
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
	c := &Config{
		Contexts: make(map[string]ContextConfig),
	}
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

func (pc Stage) GetDependsOn() (deps []string) {
	return util.ReadStringsSlice(pc.DependsOn)
}

func (pc Stage) Name() string {
	if pc.StageName != "" {
		return pc.StageName
	}

	return pc.Task
}
