package config

import (
	"fmt"
	"path/filepath"

	"github.com/taskctl/taskctl/pkg/utils"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/scheduler"
	"github.com/taskctl/taskctl/pkg/task"
)

func buildPipeline(g *scheduler.ExecutionGraph, stages []*stageDefinition, cfg *Config) (*scheduler.ExecutionGraph, error) {
	for _, def := range stages {
		var stageTask *task.Task
		var stagePipeline *scheduler.ExecutionGraph

		if def.Task != "" {
			stageTask = cfg.Tasks[def.Task]
			if stageTask == nil {
				return nil, fmt.Errorf("stage build failed: no such task %s", def.Task)
			}
		} else {
			stagePipeline = cfg.Pipelines[def.Pipeline]
			if stagePipeline == nil {
				return nil, fmt.Errorf("stage build failed: no such pipeline %s", def.Task)
			}
		}

		dir := def.Dir
		if def.Dir == "" && stageTask != nil {
			dir = stageTask.Dir
		}

		envs := variables.FromMap(def.Env)
		if def.EnvFile != "" {
			filename := def.EnvFile
			if !filepath.IsAbs(filename) && def.Dir != "" {
				filename = filepath.Join(def.Dir, filename)
			}

			fileEnvs, err := utils.ReadEnvFile(filename)
			if err != nil {
				return nil, err
			}

			envs = variables.FromMap(fileEnvs).Merge(envs)
		}

		stage := &scheduler.Stage{
			Name:         def.Name,
			Condition:    def.Condition,
			Task:         stageTask,
			Pipeline:     stagePipeline,
			DependsOn:    def.DependsOn,
			Dir:          dir,
			AllowFailure: def.AllowFailure,
			Env:          envs,
			Variables:    variables.FromMap(def.Variables),
		}

		if stage.Name == "" {
			if def.Task != "" {
				stage.Name = def.Task
			}

			if def.Pipeline != "" {
				stage.Name = def.Pipeline
			}

			if stage.Name == "" {
				return nil, fmt.Errorf("stage for task %s must have name", def.Task)
			}
		}

		stage.Variables.Set(".Stage.Name", stage.Name)

		if _, err := g.Node(stage.Name); err == nil {
			return nil, fmt.Errorf("stage with same name %s already exists", stage.Name)
		}

		err := g.AddStage(stage)
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}
