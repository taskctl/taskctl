package config

import (
	"time"

	"github.com/taskctl/taskctl/pkg/utils"
)

type configDefinition struct {
	Import    []string
	Contexts  map[string]*contextDefinition
	Pipelines map[string][]*StageDefinition
	Tasks     map[string]*TaskDefinition
	Watchers  map[string]*WatcherDefinition

	Shell utils.Binary

	Debug, DryRun bool
	Output        string

	Variables map[string]string
}

type StageDefinition struct {
	Name         string
	Condition    string
	Task         string
	Pipeline     string
	DependsOn    []string `mapstructure:"depends_on"`
	AllowFailure bool     `mapstructure:"allow_failure"`
	Dir          string
	Env          map[string]string
	Variables    map[string]string
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
	Interactive  bool
	ExportAs     string
	Env          map[string]string
	Variables    map[string]string
}

type WatcherDefinition struct {
	Events    []string
	Watch     []string
	Exclude   []string
	Task      string
	Variables map[string]string
}
