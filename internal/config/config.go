package config

import (
	"fmt"
	"os"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/watch"
	"github.com/taskctl/taskctl/pkg/output"
	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/scheduler"
	"github.com/taskctl/taskctl/pkg/task"
)

// DefaultFileNames is default names for tasks' files
var DefaultFileNames = []string{"taskctl.yaml", "tasks.yaml"}

// NewConfig creates new config instance
func NewConfig() *Config {
	cfg := &Config{
		Output:    output.FormatPrefixed,
		Contexts:  make(map[string]*runner.ExecutionContext),
		Pipelines: make(map[string]*scheduler.ExecutionGraph),
		Tasks:     make(map[string]*task.Task),
		Watchers:  make(map[string]*watch.Watcher),
		Variables: defaultConfigVariables(),
	}

	return cfg
}

// Config is a taskctl internal config structure
type Config struct {
	Import    []string
	Contexts  map[string]*runner.ExecutionContext
	Pipelines map[string]*scheduler.ExecutionGraph
	Tasks     map[string]*task.Task
	Watchers  map[string]*watch.Watcher

	Debug, DryRun, Summary bool
	Output                 string

	Variables variables.Container
}

func (cfg *Config) merge(src *Config) error {
	defer func() {
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()

	if err := mergo.Merge(cfg, src); err != nil {
		return err
	}

	return nil
}

func buildFromDefinition(def *configDefinition) (cfg *Config, err error) {
	cfg = NewConfig()

	for k, v := range def.Contexts {
		cfg.Contexts[k], err = buildContext(v)
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
			return nil, fmt.Errorf("no such task %s", v.Task)
		}
		cfg.Watchers[k], err = buildWatcher(k, v, cfg)
		if err != nil {
			return nil, err
		}
	}

	// to allow pipeline-to-pipeline links
	for k := range def.Pipelines {
		cfg.Pipelines[k] = &scheduler.ExecutionGraph{}
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
	cfg.Variables = cfg.Variables.Merge(variables.FromMap(def.Variables))

	return cfg, nil
}

func defaultConfigVariables() variables.Container {
	return variables.FromMap(map[string]string{
		"TempDir": os.TempDir(),
	})
}
