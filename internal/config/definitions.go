package config

import (
	"time"
)

type configDefinition struct {
	Import    []string
	Contexts  map[string]*contextDefinition
	Pipelines map[string][]*stageDefinition
	Tasks     map[string]*taskDefinition
	Watchers  map[string]*watcherDefinition

	Debug, DryRun, Summary bool
	Output                 string

	Variables map[string]string
}

type stageDefinition struct {
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

type taskDefinition struct {
	Name         string
	Description  string
	Condition    string
	Command      []string
	After        []string
	Before       []string
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

type watcherDefinition struct {
	Events    []string
	Watch     []string
	Exclude   []string
	Task      string
	Variables map[string]string
}
