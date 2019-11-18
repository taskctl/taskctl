package scheduler

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
)

type Pipeline struct {
	nodes map[string]*Stage
	from  map[string][]string
	to    map[string][]string
	env   map[string][]string
}

type Stage struct {
	Name      string
	Task      task.Task
	DependsOn []string
	Env       map[string]string
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
			return nil, errors.New(fmt.Sprintf("unknown task %s", def.Task))
		}

		stage := Stage{
			Name:      def.Name(),
			Task:      *t,
			DependsOn: def.GetDependsOn(),
			Env:       def.Env,
		}

		if _, ok := p.nodes[def.Name()]; ok {
			return nil, errors.New(fmt.Sprintf("stage with same name %s already exists", def.Name()))
		}

		p.addNode(def.Name(), stage)

		for _, dep := range stage.DependsOn {
			if _, ok := p.nodes[dep]; !ok {
				return nil, errors.New(fmt.Sprintf("stage %s depends on unknown stage %s", stage.Name, dep))
			}
			err := p.addEdge(dep, def.Name())
			if err != nil {
				return nil, err
			}
		}

		p.env[def.Name()] = util.ConvertEnv(stage.Env)

	}

	return p, nil
}

func (p *Pipeline) addNode(name string, stage Stage) {
	p.nodes[name] = &stage
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

func (p *Pipeline) Node(name string) *Stage {
	t, ok := p.nodes[name]
	if !ok {
		log.Fatalf("unknown task name %s\r\n", name)
	}

	return t
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
