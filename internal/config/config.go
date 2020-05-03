package config

import (
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/util"
)

const (
	ContextTypeLocal     = "local"
	ContextTypeContainer = "container"
	ContextTypeRemote    = "remote"

	FlavorRaw       = "raw"
	FlavorFormatted = "formatted"
	FlavorCockpit   = "cockpit"
)

var DefaultFileNames = []string{"taskctl.yaml", "tasks.yaml"}

var values *Config

func Get() *Config {
	return values
}

type Config struct {
	Import    []string
	Contexts  map[string]*ContextDefinition
	Pipelines map[string][]*StageDefinition
	Tasks     map[string]*TaskDefinition
	Watchers  map[string]*WatcherDefinition

	Shell         util.Executable
	Docker        util.Executable
	DockerCompose util.Executable `mapstructure:"docker-compose"`
	Kubectl       util.Executable
	Ssh           util.Executable

	Debug, DryRun bool
	Output        string

	Variables Set
}

func defaultConfig() *Config {
	return &Config{
		Output: FlavorFormatted,
	}
}

func (c *Config) merge(src *Config) error {
	defer func() {
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()

	if err := mergo.Merge(c, src); err != nil {
		return err
	}

	return nil
}

func (c *Config) init() {
	c.Output = FlavorFormatted

	for name, v := range c.Tasks {
		if v.Name == "" {
			v.Name = name
		}
	}

	if c.Contexts == nil {
		c.Contexts = make(map[string]*ContextDefinition)
	}

	if _, ok := c.Contexts[ContextTypeLocal]; !ok {
		c.Contexts[ContextTypeLocal] = &ContextDefinition{Type: ContextTypeLocal}
	}

	if c.Variables == nil {
		c.Variables = make(map[string]string)
	}
}
