package pipeline

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/taskctl/taskctl/internal/config"

	"github.com/taskctl/taskctl/internal/task"
)

type ExecutionGraph struct {
	Env map[string][]string

	nodes map[string]*Stage
	from  map[string][]string
	to    map[string][]string
	error error
}

func BuildPipeline(stages []*config.StageDefinition, pipelines map[string][]*config.StageDefinition, tasks map[string]*config.TaskDefinition) (g *ExecutionGraph, err error) {
	g = &ExecutionGraph{
		nodes: make(map[string]*Stage),
		from:  make(map[string][]string),
		to:    make(map[string][]string),
	}

	for _, def := range stages {
		var stageTask *task.Task
		var stagePipeline *ExecutionGraph

		if def.Task != "" {
			stageTaskDef, ok := tasks[def.Task]
			if !ok {
				return nil, fmt.Errorf("unknown task %s", def.Task)
			}

			stageTask = task.BuildTask(stageTaskDef)
		} else if def.Pipeline != "" {
			stagePipelineDef, ok := pipelines[def.Pipeline]
			if !ok {
				return nil, fmt.Errorf("unknown pipeline %s", def.Task)
			}

			stagePipeline, err = BuildPipeline(stagePipelineDef, pipelines, tasks)
			if err != nil {
				return nil, err
			}
		}

		stage := &Stage{
			Name:         def.Name,
			Condition:    def.Condition,
			Task:         stageTask,
			Pipeline:     stagePipeline,
			DependsOn:    def.DependsOn,
			Env:          def.Env,
			Dir:          def.Dir,
			AllowFailure: def.AllowFailure,
			Variables:    def.Variables,
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

		if _, ok := g.nodes[stage.Name]; ok {
			return nil, fmt.Errorf("stage with same name %s already exists", stage.Name)
		}

		g.addNode(stage.Name, stage)

		for _, dep := range stage.DependsOn {
			err := g.addEdge(dep, stage.Name)
			if err != nil {
				return nil, err
			}
		}
	}

	return g, nil
}

func (g *ExecutionGraph) addNode(name string, stage *Stage) {
	g.nodes[name] = stage
}

func (g *ExecutionGraph) addEdge(from string, to string) error {
	g.from[from] = append(g.from[from], to)
	g.to[to] = append(g.to[to], from)

	if err := g.cycleDfs(to, make(map[string]bool)); err != nil {
		return err
	}

	return nil
}

func (g *ExecutionGraph) Nodes() map[string]*Stage {
	return g.nodes
}

func (g *ExecutionGraph) Node(name string) (*Stage, error) {
	t, ok := g.nodes[name]
	if !ok {
		return nil, fmt.Errorf("unknown task %s", name)
	}

	return t, nil
}

func (g *ExecutionGraph) From(name string) []string {
	return g.from[name]
}

func (g *ExecutionGraph) To(name string) []string {
	return g.to[name]
}

func (g *ExecutionGraph) cycleDfs(t string, visited map[string]bool) error {
	if visited[t] {
		return errors.New("cycle detected")
	}
	visited[t] = true

	for _, next := range g.from[t] {
		err := g.cycleDfs(next, visited)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *ExecutionGraph) Error() error {
	return g.error
}

func (g *ExecutionGraph) provideOutput(s *Stage) error {
	for _, dep := range s.DependsOn {
		n, err := g.Node(dep)
		if err != nil {
			return err
		}

		if n.Task == nil {
			continue
		}

		var varName, envVarName string
		if n.Task.ExportAs == "" {
			varName = fmt.Sprintf("Output%s", strings.Title(dep))
			envVarName = fmt.Sprintf("%s_OUTPUT", strings.ToUpper(dep))
			envVarName = regexp.MustCompile("[^a-zA-Z0-9_]").ReplaceAllString(envVarName, "_")
		} else {
			varName = n.Task.ExportAs
			envVarName = n.Task.ExportAs
		}

		s.SetEnvVariable(envVarName, n.Task.Log.Stdout.String())
		s.Variables.Set(varName, n.Task.Log.Stdout.String())
	}

	return nil
}
