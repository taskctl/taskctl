package config

import (
	"fmt"
	"os"

	"github.com/Ensono/taskctl/pkg/variables"

	"dario.cat/mergo"
	"github.com/sirupsen/logrus"

	"github.com/Ensono/taskctl/internal/watch"
	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
)

// DefaultFileNames is default names for tasks' files
var DefaultFileNames = []string{"taskctl.yaml", "tasks.yaml"}

// NewConfig creates new config instance
func NewConfig() *Config {
	cfg := &Config{
		Output:    output.PrefixedOutput,
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

	Quiet, Debug, DryRun, Summary bool
	Output                        output.OutputEnum

	Variables variables.Container
	// Options are computed cli or other API inputs
	//
	Options struct {
		GraphOrientationLeftRight bool
		InitDir                   string
		InitNoPrompt              bool
	}
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

func buildFromDefinition(def *ConfigDefinition, lc *loaderContext) (cfg *Config, err error) {
	cfg = NewConfig()

	for k, v := range def.Contexts {
		cfg.Contexts[k], err = buildContext(v)
		if err != nil {
			return nil, err
		}
	}

	for k, v := range def.Tasks {
		// need to project the name from the key if not set by user
		if v.Name == "" {
			v.Name = k
		}
		builtTask, err := buildTask(v, lc)
		if err != nil {
			return nil, err
		}
		cfg.Tasks[k] = builtTask
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
		cfg.Pipelines[k], err = scheduler.NewExecutionGraph()
		if err != nil {
			return nil, err
		}
	}

	for k, v := range def.Pipelines {
		// This never errors out on the cyclical dependency
		//
		cfg.Pipelines[k], err = buildPipeline(cfg.Pipelines[k], v, cfg)
		if err != nil {
			return nil, err
		}
	}

	cfg.Import = def.Import
	cfg.Debug = def.Debug
	cfg.Output = output.OutputEnum(def.Output)
	cfg.Variables = cfg.Variables.Merge(variables.FromMap(def.Variables))

	return cfg, nil
}

func defaultConfigVariables() variables.Container {
	return variables.FromMap(map[string]string{
		"TempDir": os.TempDir(),
	})
}
