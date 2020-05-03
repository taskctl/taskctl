package config

import (
	"time"

	"github.com/taskctl/taskctl/internal/util"
)

type ContextDefinition struct {
	Type      string
	Dir       string
	Container ContainerDefinition
	SSH       SSHConfigDefinition
	Up        []string
	Down      []string
	Before    []string
	After     []string
	Env       Set
	Variables Set
	util.Executable
}

type StageDefinition struct {
	Name         string
	Condition    string
	Task         string
	Pipeline     string
	DependsOn    []string `mapstructure:"depends_on"`
	AllowFailure bool     `mapstructure:"allow_failure"`
	Dir          string
	Env          Set
	Variables    Set
}

type TaskDefinition struct {
	Name         string
	Description  string
	Condition    string
	Command      []string
	After        []string
	Context      string
	Variations   []map[string]string `yaml:",omitempty"`
	Dir          string
	Timeout      *time.Duration `yaml:",omitempty"`
	AllowFailure bool           `mapstructure:"allow_failure"`
	ExportAs     string
	Env          Set
	Variables    Set
}

type WatcherDefinition struct {
	Events    []string
	Watch     []string
	Exclude   []string
	Task      string
	Variables Set
}

type ContainerDefinition struct {
	Provider string
	Name     string
	Image    string
	Exec     bool
	Options  []string
	Env      Set
	util.Executable
}

type SSHConfigDefinition struct {
	Options []string
	User    string
	Host    string
	util.Executable
}
