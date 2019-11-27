package scheduler

import (
	"errors"
	"fmt"
	"github.com/trntv/wilson/internal/config"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
)

type Pipeline struct {
	nodes map[string]*Stage
	from  map[string][]string
	to    map[string][]string
	env   map[string][]string
}

func BuildPipeline(stages []config.Stage, tasks map[string]*task.Task) (*Pipeline, error) {
	var p = &Pipeline{
		nodes: make(map[string]*Stage),
		from:  make(map[string][]string),
		to:    make(map[string][]string),
		env:   make(map[string][]string),
	}

	for _, def := range stages {
		t := tasks[def.Task]
		if t == nil {
			return nil, fmt.Errorf("unknown task %s", def.Task)
		}

		stage := &Stage{
			Name:         def.Name,
			Task:         *t,
			Pipeline:     def.Pipeline,
			DependsOn:    def.DependsOn,
			Env:          def.Env,
			AllowFailure: def.AllowFailure,
		}

		if stage.Name == "" {
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
