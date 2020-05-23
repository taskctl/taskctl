package scheduler

import (
	"errors"
	"fmt"
	"time"
)

// ErrCycleDetected occurs when added edge causes cycle to appear
var ErrCycleDetected = errors.New("cycle detected")

// ExecutionGraph is a DAG whose nodes are Stages and edges are their dependencies
type ExecutionGraph struct {
	Env map[string][]string

	nodes      map[string]*Stage
	from       map[string][]string
	to         map[string][]string
	error      error
	start, end time.Time
}

// NewExecutionGraph creates new ExecutionGraph instance.
// It accepts zero or more stages and adds them to resulted graph
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

// AddStage adds Stage to ExecutionGraph.
// If newly added stage causes a cycle to appear in the graph it return an error
func (g *ExecutionGraph) AddStage(stage *Stage) error {
	g.addNode(stage.Name, stage)
	for _, dep := range stage.DependsOn {
		err := g.addEdge(dep, stage.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// addNode adds a new node to the graph
func (g *ExecutionGraph) addNode(name string, stage *Stage) {
	g.nodes[name] = stage
}

// addEdge adds a new edge to the graph
func (g *ExecutionGraph) addEdge(from string, to string) error {
	g.from[from] = append(g.from[from], to)
	g.to[to] = append(g.to[to], from)

	if err := g.cycleDfs(to, make(map[string]bool)); err != nil {
		return err
	}

	return nil
}

// Nodes returns ExecutionGraph stages
func (g *ExecutionGraph) Nodes() map[string]*Stage {
	return g.nodes
}

// Node returns stage by its name
func (g *ExecutionGraph) Node(name string) (*Stage, error) {
	t, ok := g.nodes[name]
	if !ok {
		return nil, fmt.Errorf("unknown task %s", name)
	}

	return t, nil
}

// From returns stages that depend on the given stage
func (g *ExecutionGraph) From(name string) []string {
	return g.from[name]
}

// To returns stages on whiсh given stage depends on
func (g *ExecutionGraph) To(name string) []string {
	return g.to[name]
}

func (g *ExecutionGraph) cycleDfs(t string, visited map[string]bool) error {
	if visited[t] {
		return ErrCycleDetected
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

// LastError returns latest error appeared during stages execution
func (g *ExecutionGraph) LastError() error {
	return g.error
}

// Duration returns execution duration
func (g *ExecutionGraph) Duration() time.Duration {
	if g.end.IsZero() {
		return time.Since(g.start)
	}

	return g.end.Sub(g.start)
}
