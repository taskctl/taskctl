package scheduler

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
)

type Pipeline struct {
	nodes map[string]*task.Task
	from  map[string][]string
	to    map[string][]string
	env   map[string][]string
}

func BuildPipeline(stages []*config.PipelineConfig, tasks map[string]*task.Task) *Pipeline {
	var graph = &Pipeline{
		nodes: make(map[string]*task.Task),
		from:  make(map[string][]string),
		to:    make(map[string][]string),
		env:   make(map[string][]string),
	}

	for _, stage := range stages {
		t := tasks[stage.Task]
		if t == nil {
			log.Fatalf("unknown task %s", stage.Task)
		}

		graph.addNode(stage.Task, t)

		for _, dep := range stage.GetDependsOn() {
			graph.addEdge(dep, stage.Task)
		}

		graph.env[stage.Task] = util.ConvertEnv(stage.Env)

		if err := graph.dfs(stage.Task, make(map[string]bool)); err != nil {
			log.Fatal("cyclic graph")
		}
	}

	return graph
}

func (p *Pipeline) addNode(name string, task *task.Task) {
	p.nodes[name] = task
}

func (p *Pipeline) addEdge(from string, to string) {
	// todo: Ensure from doesn't violates DAG constraints
	p.from[from] = append(p.from[from], to)
	p.to[to] = append(p.to[to], from)
}

func (p *Pipeline) Nodes() map[string]*task.Task {
	return p.nodes
}

func (p *Pipeline) Node(name string) *task.Task {
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

func (p *Pipeline) dfs(t string, visited map[string]bool) error {
	if visited[t] == true {
		return errors.New("cycle detected")
	}
	visited[t] = true

	for _, next := range p.from[t] {
		err := p.dfs(next, visited)
		if err != nil {
			return err
		}
	}

	return nil
}
