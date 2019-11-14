package config

import (
	"fmt"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
)

const (
	// todo: local, container, remote + providers
	CONTEXT_TYPE_LOCAL     = "local"
	CONTEXT_TYPE_CONTAINER = "docker-compose"
	//CONTEXT_TYPE_DOCKER = "docker"
	//CONTEXT_TYPE_KUBECTL = "kubectl"
	//CONTEXT_TYPE_SSH = "ssh"
)

type Executable struct {
	Bin  string
	Args []string
}

type WilsonConfig struct {
	Shell         Executable `yaml:"shell"`
	DockerCompose Executable `yaml:"docker-compose"`
	LogOnError    bool
}

type ContextConfig struct {
	Type           string
	Executable     Executable
	ComposeService struct {
		Name           string
		Transient      bool
		StartupCommand string
		CleanupCommand string
		Args           []string
		File           string
	} `yaml:"service"`
	Container struct {
		Image string
		Args  []string
	}
	Env map[string]string
}

type TaskConfig struct {
	Command []string
	Context string
	Env     map[string]string
	Dir     string
}

type PipelineConfig struct {
	Task    string
	Depends []string
}

type WatcherConfig struct {
	Events []string
	Watch  []string
	Task   string
}

type Config struct {
	Import    []string
	Contexts  map[string]*ContextConfig
	Pipelines map[string][]*PipelineConfig
	Tasks     map[string]*TaskConfig
	Watchers  map[string]*WatcherConfig

	WilsonConfig WilsonConfig
}

func Load(file string) (*Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		logrus.Fatalln(err)
	}

	c, err := load(dir, file)
	if err != nil {
		return nil, err
	}

	if _, ok := c.Contexts[CONTEXT_TYPE_LOCAL]; !ok {
		c.Contexts[CONTEXT_TYPE_LOCAL] = &ContextConfig{Type: CONTEXT_TYPE_LOCAL}
	}

	return c, nil
}

func load(dir string, file string) (*Config, error) {
	configPath := path.Join(dir, file)
	config, err := readFile(configPath)
	if err != nil {
		return nil, err
	}

	importDir := path.Dir(configPath)
	for _, file := range config.Import {
		lconfig, err := load(importDir, file)
		if err != nil {
			return nil, err
		}
		lconfig.Merge(config)
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
		return nil, err
	}

	err = yaml.UnmarshalStrict(data, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Merge(src *Config) {
	if err := mergo.Merge(c, src); err != nil {
		logrus.Fatal(err)
	}
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
