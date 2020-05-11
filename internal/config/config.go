package config

import (
	"fmt"

	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/pipeline"
	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/watch"

	"github.com/taskctl/taskctl/internal/utils"
)

// Default names for tasks' files
var DefaultFileNames = []string{"taskctl.yaml", "tasks.yaml"}

// Creates new config instance
func NewConfig() *Config {
	cfg := &Config{
		Output:    output.OutputFormatPrefixed,
		Contexts:  make(map[string]*context.ExecutionContext),
		Pipelines: make(map[string]*pipeline.ExecutionGraph),
		Tasks:     make(map[string]*task.Task),
		Watchers:  make(map[string]*watch.Watcher),
	}

	return cfg
}

type Config struct {
	Import    []string
	Contexts  map[string]*context.ExecutionContext
	Pipelines map[string]*pipeline.ExecutionGraph
	Tasks     map[string]*task.Task
	Watchers  map[string]*watch.Watcher

	Debug, DryRun bool
	Output        string

	Variables *utils.Variables
}

func (cfg *Config) Task(name string) *task.Task {
	return cfg.Tasks[name]
}

func (cfg *Config) Pipeline(name string) *pipeline.ExecutionGraph {
	return cfg.Pipelines[name]
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

func buildFromDefinition(def *configDefinition) (cfg *Config, err error) {
	cfg = NewConfig()

	for k, v := range def.Contexts {
		cfg.Contexts[k], err = buildContext(v, def.Shell)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range def.Tasks {
		cfg.Tasks[k], err = buildTask(v)
		if cfg.Tasks[k].Name == "" {
			cfg.Tasks[k].Name = k
		}
		if err != nil {
			return nil, err
		}
	}

	for k, v := range def.Watchers {
		t := cfg.Tasks[v.Task]
		if t == nil {
			return nil, fmt.Errorf("no such task")
		}
		cfg.Watchers[k], err = buildWatcher(k, v, cfg)
		if err != nil {
			return nil, err
		}
	}

	// to allow pipeline-to-pipeline links
	for k := range def.Pipelines {
		cfg.Pipelines[k] = &pipeline.ExecutionGraph{}
	}

	for k, v := range def.Pipelines {
		cfg.Pipelines[k], err = buildPipeline(v, cfg)
		if err != nil {
			return nil, err
		}
	}

	cfg.Import = def.Import
	cfg.Debug = def.Debug
	cfg.Output = def.Output
	cfg.Variables = utils.NewVariables(def.Variables)

	return cfg, nil
}
