package scheduler

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Ensono/taskctl/internal/utils"
)

var (
	// ErrCycleDetected occurs when added edge causes cycle to appear
	ErrCycleDetected = errors.New("cycle detected")
	// ErrNodeNotFound occurs when node is not found in the graph
	ErrNodeNotFound = errors.New("node not found")
	ErrRunTimeFault = errors.New("task execution fault")
)

const (
	RootNodeName = "root"
)

type GraphError struct {
	stage *Stage
	err   error
}

// ExecutionGraph is a DAG whose nodes are Stages and edges are their dependencies
type ExecutionGraph struct {
	errors    []GraphError
	Generator map[string]any
	Env       map[string]string
	EnvFile   *utils.Envfile
	name      string
	alias     string
	nodes     map[string]*Stage
	// parent holds the children reference of the node
	parent map[string][]string
	// children points back children the parent reference
	children   map[string][]string
	start, end time.Time
	mu         sync.Mutex
}

// NewExecutionGraph creates new ExecutionGraph instance.
// It accepts zero or more stages and adds them to resulted graph
func NewExecutionGraph(name string, stages ...*Stage) (*ExecutionGraph, error) {
	// create a rooted node to hang the graph of
	// this will allow for easy find of the initial node(s)
	rootNode := NewStage(name, func(s *Stage) {
		s.Name = RootNodeName
	})
	rootNode.UpdateStatus(StatusDone)

	nodes := map[string]*Stage{RootNodeName: rootNode}

	graph := &ExecutionGraph{
		errors:   []GraphError{},
		nodes:    nodes,
		name:     name,
		parent:   make(map[string][]string),
		children: make(map[string][]string),
	}

	for _, stage := range stages {
		if err := graph.AddStage(stage); err != nil {
			return nil, err
		}
	}

	return graph, nil
}

func (g *ExecutionGraph) WithAlias(v string) *ExecutionGraph {
	g.alias = v
	return g
}

// VisitNodes visits all nodes in a given graph
// or recursively through all subgraphs in a parented graph
func (g *ExecutionGraph) VisitNodes(callback func(node *Stage) (done bool), recursive bool) {
	for _, node := range g.Nodes() {
		if recursive {
			if node.Pipeline != nil {
				node.Pipeline.VisitNodes(callback, true)
			}
		}
		done := callback(node)
		if done {
			return
		}
	}
}

// AddStage adds Stage to ExecutionGraph.
// If newly added stage causes a cycle to appear in the graph it return an error
func (g *ExecutionGraph) AddStage(stage *Stage) error {

	g.addNode(stage.Name, stage)

	if len(stage.DependsOn) == 0 {
		return g.addEdge(RootNodeName, stage.Name)
	}
	for _, dep := range stage.DependsOn {
		err := g.addEdge(dep, stage.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// addNode adds a new node to the map (index of nodes)
func (g *ExecutionGraph) addNode(name string, stage *Stage) {
	g.nodes[name] = stage
}

// addEdge adds a new edge to the graph
// from is the child
// to is the parent of the node
func (g *ExecutionGraph) addEdge(parent string, child string) error {
	g.parent[child] = append(g.parent[child], parent)
	g.children[parent] = append(g.children[parent], child)
	return g.cycleDfs(parent, make(map[string]bool), make(map[string]bool))
}

// Nodes returns ExecutionGraph stages - an n-ary tree itself
// Stage (Node) may appear multiple times in a scheduling scenario,
// this is desired behaviour to loop over the nodes as many times
// as they appear in a DAG manner.
func (g *ExecutionGraph) Nodes() map[string]*Stage {
	return g.nodes
}

// Node returns stage by its name
func (g *ExecutionGraph) Node(name string) (*Stage, error) {
	t, ok := g.nodes[name]
	if !ok {
		return nil, fmt.Errorf("%w %s", ErrNodeNotFound, name)
	}
	return t, nil
}

// Parents returns stages on whiсh given stage depends on
func (g *ExecutionGraph) Parents(name string) map[string]*Stage {
	stages := make(map[string]*Stage)

	for _, nodeName := range g.parent[name] {
		stages[nodeName] = g.nodes[nodeName]
	}
	return stages
}

func (g *ExecutionGraph) Children(node string) map[string]*Stage {
	stages := make(map[string]*Stage)
	for _, nodeName := range g.children[node] {
		stages[nodeName] = g.nodes[nodeName]
	}
	return stages
}

// BFSNodesFlattened returns a Breadth-First-Search flattened list of top level tasks/pipelines
// This is useful in summaries as we want the things that run in parallel
// on the same level to show in that order before the level below and so on.
//
// When generating CI definitions - we don't need to generate the same jobs/steps over and over again
// they will be referenced with a needs/depends_on/etc... keyword.
//
// Returns a slice of stages in this level of the graph.
func (g *ExecutionGraph) BFSNodesFlattened(nodeName string) StageList {
	bfsStages := StageList{}
	// Create a queue to keep track of nodes to visit
	queue := []string{nodeName}
	// Create a map to keep track of visited nodes
	visited := make(map[string]bool)

	visited[nodeName] = true

	// Start the BFS loop
	for len(queue) > 0 {
		// Dequeue the first node from the queue
		current := queue[0]
		queue = queue[1:]

		// add to flattened list - except if it's the root node
		if current != RootNodeName {
			bfsStages = append(bfsStages, g.nodes[current])
		}

		// Enqueue all unvisited adjacent nodes (children)
		for _, child := range g.children[current] {
			if !visited[child] {
				queue = append(queue, child)
				visited[child] = true
			}
		}
	}
	return bfsStages
}

// cycleDfs is DFS utility to traverse
// the tree to detect any back-edges and hence to detect a cycle
func (g *ExecutionGraph) cycleDfs(node string, visited map[string]bool, inStack map[string]bool) error {
	// Mark the node as visited and part of the current recursion stack
	visited[node] = true
	inStack[node] = true

	// Explore all the children of the current node
	for _, child := range g.children[node] {
		// If the child is not visited, recurse
		if !visited[child] {
			if err := g.cycleDfs(child, visited, inStack); err != nil {
				return err
			}
		}
		// if a child is already in the stack we return a cycle is detect it
		if inStack[child] {
			return fmt.Errorf("pipeline (%s) already contains [%s] -> [%s] - reversing it would create a cyclical dependency\n%w", g.name, child, node, ErrCycleDetected)
		}
	}

	// Remove the node from the recursion stack after processing
	inStack[node] = false
	return nil
}

func (g *ExecutionGraph) WithStageError(stage *Stage, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.errors = append(g.errors, GraphError{stage: stage, err: err})
}

// LastError returns latest error appeared during stages execution
func (g *ExecutionGraph) LastError() error {
	if len(g.errors) > 0 {
		return g.errors[len(g.errors)-1].err
	}
	return nil
}

func (g *ExecutionGraph) Error() error {
	if len(g.errors) > 0 {
		es := ""
		for _, v := range g.errors {
			es += fmt.Sprintf("stage: %s\nerror: %v\n", v.stage.Name, v.err)
		}
		return fmt.Errorf("%w, %s", ErrRunTimeFault, es)
	}
	return nil
}

// Name returns the name of the graph
func (g *ExecutionGraph) Name() string {
	return g.name
}

// Duration returns execution duration
func (g *ExecutionGraph) Duration() time.Duration {
	if g.end.IsZero() {
		return time.Since(g.start)
	}

	return g.end.Sub(g.start)
}
