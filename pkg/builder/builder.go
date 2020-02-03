package builder

import (
	"github.com/taskctl/taskctl/pkg/util"
	"time"
)

type TaskctlConfigDefinition struct {
	Shell         util.Executable
	Docker        util.Executable
	DockerCompose util.Executable `mapstructure:"docker-compose"`
	Kubectl       util.Executable
	Ssh           util.Executable
	Debug         bool
}

type ContextDefinition struct {
	Type      string
	Dir       string
	Container ContainerDefinition
	SSH       SSHConfigDefinition
	Env       map[string]string
	Up        []string
	Down      []string
	Before    []string
	After     []string
	util.Executable
}

type StageDefinition struct {
	Name         string
	Task         string
	Pipeline     string
	DependsOn    []string `mapstructure:"depends_on"`
	Env          map[string]string
	AllowFailure bool `mapstructure:"allow_failure"`
}

type TaskDefinition struct {
	Name         string
	Description  string
	Command      []string
	After        []string
	Context      string
	Env          map[string]string   `yaml:",omitempty"`
	Variations   []map[string]string `yaml:",omitempty"`
	Dir          string
	Timeout      *time.Duration `yaml:",omitempty"`
	AllowFailure bool           `mapstructure:"allow_failure"`
}

type WatcherDefinition struct {
	Events  []string
	Watch   []string
	Exclude []string
	Task    string
}

type ContainerDefinition struct {
	Provider string
	Name     string
	Image    string
	Exec     bool
	Options  []string
	Env      map[string]string
	util.Executable
}

type SSHConfigDefinition struct {
	Options []string
	User    string
	Host    string
	util.Executable
}
