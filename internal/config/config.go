package config

import (
	"io"

	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/output"
	"github.com/taskctl/taskctl/pkg/util"

	"github.com/taskctl/taskctl/pkg/builder"
)

const (
	ContextTypeLocal     = "local"
	ContextTypeContainer = "container"
	ContextTypeRemote    = "remote"

	ContextContainerProviderDocker        = "docker"
	ContextContainerProviderDockerCompose = "docker-compose"
	ContextContainerProviderKubectl       = "kubectl"
)

var DefaultFileNames = []string{"taskctl.yaml", "tasks.yaml"}

var values *Config

func Get() *Config {
	return values
}

type Config struct {
	Import    []string
	Contexts  map[string]*builder.ContextDefinition
	Pipelines map[string][]*builder.StageDefinition
	Tasks     map[string]*builder.TaskDefinition
	Watchers  map[string]*builder.WatcherDefinition

	Shell         util.Executable
	Docker        util.Executable
	DockerCompose util.Executable `mapstructure:"docker-compose"`
	Kubectl       util.Executable
	Ssh           util.Executable

	configMap map[string]interface{}
	W         io.Writer

	Debug, DryRun bool
	Output        string

	Variables map[string]string
}

func defaultConfig() *Config {
	return &Config{
		Output: output.FlavorFormatted,
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
	c.Output = output.FlavorFormatted

	for name, v := range c.Tasks {
		if v.Name == "" {
			v.Name = name
		}
	}

	if c.Contexts == nil {
		c.Contexts = make(map[string]*builder.ContextDefinition)
	}

	if _, ok := c.Contexts[ContextTypeLocal]; !ok {
		c.Contexts[ContextTypeLocal] = &builder.ContextDefinition{Type: ContextTypeLocal}
	}

	if c.Variables == nil {
		c.Variables = make(map[string]string)
	}
}
