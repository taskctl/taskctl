package config

import (
	"fmt"

	"github.com/taskctl/taskctl/internal/pipeline"
	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/util"
)

func buildPipeline(stages []*StageDefinition, cfg *Config) (g *pipeline.ExecutionGraph, err error) {
	g = pipeline.NewExecutionGraph()

	for _, def := range stages {
		var stageTask *task.Task
		var stagePipeline *pipeline.ExecutionGraph

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

		stage := &pipeline.Stage{
			Name:         def.Name,
			Condition:    def.Condition,
			Task:         stageTask,
			Pipeline:     stagePipeline,
			DependsOn:    def.DependsOn,
			Dir:          def.Dir,
			AllowFailure: def.AllowFailure,
			Env:          util.NewVariables(def.Env),
			Variables:    util.NewVariables(def.Variables),
		}

		if stage.Dir != "" {
			stage.Task.Dir = stage.Dir
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

		g.AddNode(stage.Name, stage)

		for _, dep := range stage.DependsOn {
			err := g.AddEdge(dep, stage.Name)
			if err != nil {
				return nil, err
			}
		}
	}

	return g, nil
}
