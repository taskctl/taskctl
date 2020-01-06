package config

import (
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/builder"
)

const (
	ContextTypeLocal     = "local"
	ContextTypeContainer = "container"
	ContextTypeRemote    = "remote"

	ContextContainerProviderDocker        = "docker"
	ContextContainerProviderDockerCompose = "docker-compose"
	ContextContainerProviderKubectl       = "kubectl"
)

var cfg *Config

func Get() *Config {
	return cfg
}

type Config struct {
	Import    []string
	Contexts  map[string]*builder.ContextDefinition
	Pipelines map[string][]*builder.StageDefinition
	Tasks     map[string]*builder.TaskDefinition
	Watchers  map[string]*builder.WatcherDefinition

	builder.WilsonConfigDefinition `mapstructure:",squash"`

	configMap map[string]interface{}
}

func (c *Config) merge(src *Config) error {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()

	if err := mergo.Merge(c, src); err != nil {
		return err
	}

	return nil
}

func (c *Config) init() {
	for name, v := range c.Tasks {
		if v.Name == "" {
			v.Name = name
		}
	}

	if _, ok := c.Contexts[ContextTypeLocal]; !ok {
		c.Contexts[ContextTypeLocal] = &builder.ContextDefinition{Type: ContextTypeLocal}
	}
}
