package scheduler

import (
	"slices"
	"strings"

	"github.com/Ensono/taskctl/internal/utils"
)

// StageTable is a simple hash table of denormalized stages into a flat hash table (map)
//
// NOTE: used for read only at this point
type StageTable map[string]*Stage

// Denormalize performs a recursive DFS traversal on the ExecutionGraph from the root node and creates a new stage reference.
//
// In order to be able to call the same pipeline from another pipeline, we want to create a new
// pointer to it, this will avoid race conditions in times/outputs/env vars/etc...
// We can also set separate vars and environment variables
//
// The denormalized pipeline will include all the same stages and nested pipelines,
// but with all names rebuilt using the cascaded ancestors as prefixes
func (g *ExecutionGraph) Denormalize() (*ExecutionGraph, error) {
	dg, _ := NewExecutionGraph(g.Name())
	flattenedStages := map[string]*Stage{}

	g.Flatten(RootNodeName, []string{g.Name()}, flattenedStages)
	// rebuild graph from flatten denormalized stages
	if err := dg.rebuildFromDenormalized(StageTable(flattenedStages)); err != nil {
		return nil, err
	}
	return dg, nil
}

// rebuildFromDenormalized rebuilds the whole tree from scratch using the denormalized stages as input
//
// Following the same layout as original with same levels of nestedness
func (g *ExecutionGraph) rebuildFromDenormalized(st StageTable) error {
	for _, stage := range st.NthLevelChildren(g.Name(), 1) {
		if stage.Pipeline != nil {
			c := st.NthLevelChildren(stage.Name, 1)
			// There is no chance that at this point there would be a cycle
			// but keep this check here just in case
			ng, err := NewExecutionGraph(stage.Name, c...)
			if err != nil {
				return err
			}
			if err := ng.rebuildFromDenormalized(st); err != nil {
				return err
			}
			stage.Pipeline = ng
		}
		// stage is task - merge into the stage all the previous env and vars
		parentStages := st.RecurseParents(stage.Name)
		for _, v := range parentStages {
			stage.Env().MergeV2(v.Env())
		}
		// Check err just in case the denormalized graph has cyclical dependancies
		if err := g.AddStage(stage); err != nil {
			// This should never be hit, but good to keep in place.
			return err
		}
	}
	return nil
}

// NthLevelChildren retrieves the nodes by prefix and depth specified
//
// removing the base prefix and looking at the depth of the keyprefix per stage
func (st StageTable) NthLevelChildren(prefix string, depth int) []*Stage {
	prefixParts := strings.Split(prefix, utils.PipelineDirectionChar)
	stages := []*Stage{}
	for key, stageVal := range st {
		if strings.HasPrefix(key, prefix) && key != prefix {
			keyParts := strings.Split(key, utils.PipelineDirectionChar)
			if len(keyParts[len(prefixParts):]) == depth {
				stages = append(stages, stageVal)
			}
		}
	}
	return stages
}

// RecurseParents walks all the parents recursively
// and appends to the list in revers order
func (st StageTable) RecurseParents(prefix string) []*Stage {
	prefixParts := strings.Split(prefix, utils.PipelineDirectionChar)
	stages := []*Stage{}
	for i := 1; i < len(prefixParts); i++ {
		parentKey := strings.Join(prefixParts[0:len(prefixParts)-i], utils.PipelineDirectionChar)
		if stageVal, ok := st[parentKey]; ok {
			stages = append(stages, stageVal)
		}
	}
	slices.Reverse(stages)
	return stages
}

// Flatten is a recursive helper function to clone nodes with unique paths.
//
// Each new instance will have a separate memory address allocation. Will be used for denormalization.
func (graph *ExecutionGraph) Flatten(nodeName string, ancestralParentNames []string, flattenedStage map[string]*Stage) {
	uniqueName := utils.CascadeName(ancestralParentNames, nodeName)
	if nodeName != RootNodeName {
		originalNode, _ := graph.Node(nodeName)
		clonedStage := NewStage(uniqueName)
		// Task or stage needs adding
		// Dereference the new stage from the original node
		clonedStage.FromStage(originalNode, graph, ancestralParentNames)
		flattenedStage[uniqueName] = clonedStage

		// If the node has a subgraph, recursively clone it with a new prefix
		if originalNode.Pipeline != nil {
			var subGraphClone *ExecutionGraph
			subGraphClone, originalNode = graphClone(originalNode, clonedStage, uniqueName)
			// use alias or name
			for subNode := range originalNode.Pipeline.Nodes() {
				originalNode.Pipeline.Flatten(subNode, append(ancestralParentNames, originalNode.Name), flattenedStage)
			}
			clonedStage.Pipeline = subGraphClone
		}
	}

	// Clone each child node, creating unique names based on the current path
	for _, child := range graph.Children(nodeName) {
		graph.Flatten(child.Name, ancestralParentNames, flattenedStage)
	}
}

func graphClone(originalNode *Stage, clonedStage *Stage, uniqueName string) (*ExecutionGraph, *Stage) {
	// creating a graph without stages - cannot error here
	subGraphClone, _ := NewExecutionGraph(uniqueName)
	// peek if children are a single pipeline
	peek := originalNode.Pipeline.Nodes()
	// its name is likely reused elsewhere
	if len(peek) == 2 { // there will always be a "root" node and 1 or more others
		for _, peekStage := range peek {
			// we skip the root node
			if peekStage.Name == RootNodeName {
				continue
			}
			if peekStage.Pipeline != nil {
				// aliased stage only contains a single item and
				// that is a pipeline we advance  move forward
				peekStage.DependsOn = clonedStage.DependsOn
				peekStage.Name = originalNode.Name
				peekStage.WithEnv(originalNode.Env())
				peekStage.WithVariables(originalNode.Variables())
				clonedStage.WithEnv(peekStage.Env())
				clonedStage.WithVariables(peekStage.Variables())
				originalNode = peekStage
			}
		}
	}
	return subGraphClone, originalNode
}
