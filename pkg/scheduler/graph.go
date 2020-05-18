package scheduler

import (
	"errors"
	"fmt"
)

type ExecutionGraph struct {
	Env map[string][]string

	nodes map[string]*Stage
	from  map[string][]string
	to    map[string][]string
	error error
}

func NewExecutionGraph(stages ...*Stage) (*ExecutionGraph, error) {
	graph := &ExecutionGraph{
		nodes: make(map[string]*Stage),
		from:  make(map[string][]string),
		to:    make(map[string][]string),
	}

	var err error
	for _, stage := range stages {
		err = graph.AddStage(stage)
		if err != nil {
			return nil, err
		}
	}

	return graph, nil
}

func (g *ExecutionGraph) AddStage(stage *Stage) error {
	g.AddNode(stage.Name, stage)
	for _, dep := range stage.DependsOn {
		err := g.AddEdge(dep, stage.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *ExecutionGraph) AddNode(name string, stage *Stage) {
	g.nodes[name] = stage
}

func (g *ExecutionGraph) AddEdge(from string, to string) error {
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
