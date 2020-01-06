package scheduler

import (
	"errors"
	"fmt"
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
)

type Pipeline struct {
	nodes map[string]*Stage
	from  map[string][]string
	to    map[string][]string
	env   map[string][]string
	error error
}

func BuildPipeline(stages []*builder.StageDefinition, pipelines map[string][]*builder.StageDefinition, tasks map[string]*builder.TaskDefinition) (p *Pipeline, err error) {
	p = &Pipeline{
		nodes: make(map[string]*Stage),
		from:  make(map[string][]string),
		to:    make(map[string][]string),
		env:   make(map[string][]string),
	}

	for _, def := range stages {
		var stageTask *task.Task
		var stagePipeline *Pipeline

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

			stagePipeline, err = BuildPipeline(stagePipelineDef, pipelines, tasks) // todo: detect cycles
			if err != nil {
				return nil, err
			}
		}

		stage := &Stage{
			Name:         def.Name,
			Task:         stageTask,
			Pipeline:     stagePipeline,
			DependsOn:    def.DependsOn,
			Env:          def.Env,
			AllowFailure: def.AllowFailure,
		}

		if stage.Name == "" {
			if def.Task != "" {
				stage.Name = def.Task
			}

			if def.Pipeline != "" {
				stage.Name = def.Pipeline
			}
			return nil, fmt.Errorf("stage for task %s must have name", def.Task)
		}

		if _, ok := p.nodes[stage.Name]; ok {
			return nil, fmt.Errorf("stage with same name %s already exists", stage.Name)
		}

		p.addNode(stage.Name, stage)

		for _, dep := range stage.DependsOn {
			err := p.addEdge(dep, stage.Name)
			if err != nil {
				return nil, err
			}
		}

		p.env[stage.Name] = util.ConvertEnv(stage.Env)

	}

	return p, nil
}

func (p *Pipeline) addNode(name string, stage *Stage) {
	p.nodes[name] = stage
}

func (p *Pipeline) addEdge(from string, to string) error {
	p.from[from] = append(p.from[from], to)
	p.to[to] = append(p.to[to], from)

	if err := p.cycleDfs(to, make(map[string]bool)); err != nil {
		return err
	}

	return nil
}

func (p *Pipeline) Nodes() map[string]*Stage {
	return p.nodes
}

func (p *Pipeline) Node(name string) (*Stage, error) {
	t, ok := p.nodes[name]
	if !ok {
		return nil, fmt.Errorf("unknown task %s", name)
	}

	return t, nil
}

func (p *Pipeline) From(name string) []string {
	return p.from[name]
}

func (p *Pipeline) To(name string) []string {
	return p.to[name]
}

func (p *Pipeline) cycleDfs(t string, visited map[string]bool) error {
	if visited[t] == true {
		return errors.New("cycle detected")
	}
	visited[t] = true

	for _, next := range p.from[t] {
		err := p.cycleDfs(next, visited)
		if err != nil {
			return err
		}
	}

	return nil
}
