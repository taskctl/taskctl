package config

import (
	"fmt"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
)

const (
	// todo: local, container, remote + providers
	CONTEXT_TYPE_LOCAL     = "local"
	CONTEXT_TYPE_CONTAINER = "container"
	CONTEXT_TYPE_REMOTE    = "remote"

	CONTEXT_CONTAINER_PROVIDER_DOCKER         = "docker"
	CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE = "docker-compose"
	CONTEXT_CONTAINER_PROVIDER_KUBECTL        = "kubectl"
	CONTEX_REMOTE_PROVIDER_SSH                = "ssh"
)

var loaded = make(map[string]bool)

type Executable struct {
	Bin  string
	Args []string
}

type WilsonConfig struct {
	Shell         Executable `yaml:"shell"`
	DockerCompose Executable `yaml:"docker-compose"`
	Kubectl       Executable `yaml:"kubectl"`
}

type ContextConfig struct {
	Type       string
	Executable Executable
	Container  struct {
		Provider string
		Name     string
		Image    string
		Exec     bool
		Options  []string
		Env      map[string]string
	}
	Env map[string]string
}

type TaskConfig struct {
	Command []string
	Context string
	Env     map[string]string
	Dir     string
}

type WatcherConfig struct {
	Events []string
	Watch  []string
	Task   string
}

type Config struct {
	Import    []string
	Contexts  map[string]*ContextConfig
	Pipelines map[string]struct {
		Tasks []*PipelineConfig
	}
	Tasks    map[string]*TaskConfig
	Watchers map[string]*WatcherConfig

	WilsonConfig WilsonConfig
}

func Load(file string) (*Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	c, err := load(path.Join(dir, file))
	if err != nil {
		return nil, err
	}

	if _, ok := c.Contexts[CONTEXT_TYPE_LOCAL]; !ok {
		c.Contexts[CONTEXT_TYPE_LOCAL] = &ContextConfig{Type: CONTEXT_TYPE_LOCAL}
	}

	return c, nil
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
		Contexts: make(map[string]*ContextConfig),
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

func ConvertEnv(env map[string]string) []string {
	var i int
	enva := make([]string, len(env))
	for k, v := range env {
		enva[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}

	return enva
}
