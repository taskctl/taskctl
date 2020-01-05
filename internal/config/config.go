package config

import (
	"fmt"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
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
	Contexts  map[string]*builder.ContextDefinition
	Pipelines map[string][]*builder.StageDefinition
	Tasks     map[string]*builder.TaskDefinition
	Watchers  map[string]*builder.WatcherDefinition

	builder.WilsonConfigDefinition
}

func (c *Config) Set(key string, value string) error {
	err := util.SetByPath(key, value, c)
	if err != nil {
		return fmt.Errorf("error setting value for key %s: %w", key, err)
	}

	return nil
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
