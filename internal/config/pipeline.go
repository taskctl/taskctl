package config

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
)

var ErrStageBuildFailure = errors.New("stage build failed")

var forbiddenCharSequence = []string{
	utils.PipelineDirectionChar, // is used by the program to delimit nested graphs
}

func buildPipeline(g *scheduler.ExecutionGraph, stages []*PipelineDefinition, cfg *Config) (*scheduler.ExecutionGraph, error) {
	for _, def := range stages {
		var stageTask *task.Task
		var stagePipeline *scheduler.ExecutionGraph

		if def.Task != "" {
			stageTask = cfg.Tasks[def.Task]
			if stageTask == nil {
				return nil, fmt.Errorf("%w: no such task %s", ErrStageBuildFailure, def.Task)
			}
			stageTask.Generator = def.Generator
		} else {
			stagePipeline = cfg.Pipelines[def.Pipeline]
			if stagePipeline == nil {
				return nil, fmt.Errorf("%w: no such pipeline %s", ErrStageBuildFailure, def.Pipeline)
			}
			stagePipeline.Generator = def.Generator

		}
		stage := scheduler.NewStage(def.Name, func(s *scheduler.Stage) {
			s.Condition = def.Condition
			s.Task = stageTask
			s.Pipeline = stagePipeline
			s.DependsOn = def.DependsOn
			s.Dir = def.Dir
			s.AllowFailure = def.AllowFailure
			s.Generator = def.Generator
		})
		if stagePipeline != nil && def.Name != "" && def.Pipeline != def.Name {
			_ = stagePipeline.WithAlias(def.Pipeline)
			stage.Alias = def.Pipeline
		}
		stage.WithEnv(variables.FromMap(def.Env))
		vars := variables.FromMap(def.Variables)
		vars.Set(".Stage.Name", def.Name)
		stage.WithVariables(vars)

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
				return nil, fmt.Errorf("%w, stage for task %s must have name", ErrStageBuildFailure, def.Task)
			}
		}

		if slices.Contains(forbiddenCharSequence, stage.Name) {
			return nil, fmt.Errorf("%w: name (%s) contains a forbidden character [ %q ]", ErrStageBuildFailure, stage.Name, forbiddenCharSequence)
		}

		if _, err := g.Node(stage.Name); err == nil {
			return nil, fmt.Errorf("%w, stage with same name %s already exists", ErrStageBuildFailure, stage.Name)
		}

		err := g.AddStage(stage)
		if err != nil {
			return nil, err
		}
	}
	// check graph has a start node and that all children and parents exist
	rootNodes := []string{}
	var finalWalkErr error
	g.VisitNodes(func(node *scheduler.Stage) (done bool) {
		if len(node.DependsOn) > 0 {
			for _, depsOn := range node.DependsOn {
				// check if stage exists
				if _, err := g.Node(depsOn); err != nil {
					finalWalkErr = fmt.Errorf("%w, stage (%s) depends on a stage (%s) which does not exist", ErrStageBuildFailure, node.Name, depsOn)
					return true
				}
			}
			return
		}
		rootNodes = append(rootNodes, node.Name)
		return
	}, false)

	if finalWalkErr != nil {
		return nil, finalWalkErr
	}
	if len(rootNodes) == 0 {
		return nil, fmt.Errorf("%w, pipeline (%s) has no starting point (at least one stage have no 'depends_on')", ErrStageBuildFailure, g.Name())
	}

	return g, nil
}
