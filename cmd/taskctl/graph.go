package cmd

import (
	"fmt"
	"io"
	"slices"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/emicklei/dot"
	"github.com/spf13/cobra"
)

type graphFlags struct {
	leftToRight bool
	isMermaid   bool
	embedLegend bool
}

func newGraphCmd(rootCmd *TaskCtlCmd) {
	f := &graphFlags{}
	graphCmd := &cobra.Command{
		Use:     "graph",
		Aliases: []string{"g"},
		Short:   `visualizes pipeline execution graph`,
		Long: `Generates a visual representation of pipeline execution plan.
The output is in the DOT format, which can be used by GraphViz to generate charts.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			p := conf.Pipelines[args[0]]
			if p == nil {
				return fmt.Errorf("no such pipeline %s", args[0])
			}
			return graphCmdRun(p, rootCmd.ChannelOut, f)
		},
	}

	graphCmd.Flags().BoolVarP(&f.leftToRight, "lr", "", false, "orientates outputted graph left-to-right")
	_ = rootCmd.viperConf.BindPFlag("lr", graphCmd.Flags().Lookup("lr"))
	graphCmd.Flags().BoolVarP(&f.isMermaid, "mermaid", "", false, "output the graph in mermaid flowchart format")
	graphCmd.Flags().BoolVarP(&f.embedLegend, "legend", "", false, "embed a legend in the generated dotviz graph")

	rootCmd.Cmd.AddCommand(graphCmd)
}

const pipelineStartKey string = "pipeline:start"

func graphCmdRun(p *scheduler.ExecutionGraph, channelOut io.Writer, f *graphFlags) error {
	tln := []string{}
	for _, v := range p.BFSNodesFlattened(scheduler.RootNodeName) {
		tln = append(tln, v.Name)
	}

	g := dot.NewGraph(dot.Directed)
	g.Attr("center", "true")
	if f.leftToRight {
		g.Attr("rankdir", "LR")
	}
	draw(g, p, tln, pipelineStartKey)
	if f.isMermaid {
		fmt.Fprintln(channelOut, dot.MermaidFlowchart(g, dot.MermaidTopToBottom))
		return nil
	}
	if f.embedLegend {
		addLegend(g)
	}
	fmt.Fprintln(channelOut, g.String())
	return nil
}

func anchorName(v string) string {
	return fmt.Sprintf("%s_anchor", v)
}

// draw recursively walks the tree and adds nodes with a correct dependency
// between the nodes (parents => children).
//
// Same nodes can be called multiple times, this relationship of nested graphs (pipelines)
// is denoted via different relationship arrows/colours.
func draw(g *dot.Graph, p *scheduler.ExecutionGraph, topLevelStages []string, parent string) {
	for _, v := range p.BFSNodesFlattened(scheduler.RootNodeName) {
		if v.Pipeline != nil {
			// check if subgraph has been added and all it's children
			if sub := getSubGraph(g, v); sub != nil {
				linkSubGraph(g, sub, v)
			} else {
				createSubGraph(g, v, topLevelStages)
			}
		}

		annotateLeaf(g, p, v, parent)

		for _, child := range p.Children(v.Name) {
			addEdges(g, v, child, topLevelStages)
		}
	}
}

//
// Helpers for draw function
//

// annotateLeaf checks if a stage is a Task and has no parents and no children
// this would be the
func annotateLeaf(g *dot.Graph, p *scheduler.ExecutionGraph, v *scheduler.Stage, parent string) {
	if v.Task != nil && len(v.DependsOn) == 0 && len(p.Children(v.Name)) == 0 {
		// since we are flattening all subgprahs we need to create the edge from the root
		// a relationship will be drawn from the top level  stage/job to subgraph
		// so we remove the edge relationship line
		g.Root().Edge(g.Root().Node(parent), g.Node(v.Name)).Attrs("style", "invis")
		// check if task already exist in the graph
		taskNode := getNode(g.Root(), v.Name)
		// check if parent is another pipeline
		// locate the anchor of the pipeline
		parentSubgraph := getNode(g.Root(), anchorName(parent))
		if parentSubgraph != nil && taskNode != nil {
			// if parentGRaph anchor and task already exist
			// then we point to it from the subgrpah anchor
			g.Edge(*parentSubgraph, *taskNode).Attr("color", "green")
		}
	}
}

// getSubGraph
func getSubGraph(g *dot.Graph, v *scheduler.Stage) *dot.Graph {
	var sub *dot.Graph
	var found bool

	// check if subgraph has been added and all it's children
	if sub, found = g.Root().FindSubgraph(v.Pipeline.Name()); !found {
		if sub, found = g.Root().FindSubgraph(v.Name); !found {
			return nil
		}
	}
	return sub
}

// addEdges adds the task/pipeline nodes to the parent
func addEdges(g *dot.Graph, v *scheduler.Stage, child *scheduler.Stage, topLevelStages []string) {
	var edge dot.Edge
	if parent := getNode(g, v.Name); parent != nil {
		edge = g.Edge(*parent, g.Node(child.Name))
	} else {
		edge = g.Edge(g.Node(v.Name), g.Node(child.Name))
	}
	if slices.Contains(topLevelStages, v.Name) {
		edge.Attr("color", "blue")
	} else {
		edge.Attr("color", "green")
	}
}

// linkSubGraph connects an anchor to an already created subgraph.
// Creating or using an existing parentNode
func linkSubGraph(g *dot.Graph, sub *dot.Graph, v *scheduler.Stage) {
	anchorNode := getNode(sub, anchorName(v.Pipeline.Name()))
	parentNode := getNode(g, v.Name)
	if parentNode == nil {
		// create the pipeline pointer node in the subgraph
		// if it does not exist
		pn := g.Node(v.Name)
		parentNode = &pn
	}
	if anchorNode != nil && parentNode != nil {
		g.Root().Edge(*parentNode, *anchorNode).Attr("color", "brown")
	}
}

// createSubGraph adds a new subgraph to the root graph
func createSubGraph(g *dot.Graph, v *scheduler.Stage, topLevelStages []string) {
	// hoist the subgraph to the top
	cluster := g.Root().Subgraph(v.Pipeline.Name(), dot.ClusterOption{})
	anchorNode := cluster.Node(anchorName(v.Pipeline.Name())).Attr("shape", "point").Attr("style", "invis")
	// loop through subgraph - by adding edges to it
	draw(cluster, v.Pipeline, topLevelStages, v.Pipeline.Name())
	// add edge fom parent graph to subgraph cluster
	var edge dot.Edge
	if pipelineNode := getNode(g, v.Name); pipelineNode != nil {
		edge = g.Edge(*pipelineNode, anchorNode)
	} else {
		edge = g.Edge(g.Node(v.Name), anchorNode)
	}
	edge.Attr("color", "brown")
}

// getNode helper looks a node by Id and falling back on to label if not found
func getNode(g *dot.Graph, id string) *dot.Node {
	node, found := g.FindNodeById(id)
	if found {
		return &node
	}
	ln, lfound := g.FindNodeWithLabel(id)
	if lfound {
		return &ln
	}
	return nil
}

func addLegend(g *dot.Graph) {
	legend := g.Subgraph("__legend__", dot.ClusterOption{}).Label("Legend")

	legend.Attr("style", "filled")
	legend.Attr("color", "lightgrey")

	samplePipeline := legend.Subgraph("__legend____pipeline__", dot.ClusterOption{}).Label("Job 2")
	samplePipeline.Attr("style", "dashed")
	samplePipeline.Attr("color", "black")
	samplePipeline.Edge(samplePipeline.Node("job task 1"), samplePipeline.Node("job task2")).Attr("color", "green")

	anchorNode := samplePipeline.Node("__legend__pipeline_anchor").Attr("style", "invis")

	legend.Edge(legend.Node("Job 1"), legend.Node("Job 2")).Attr("color", "blue")
	legend.Edge(legend.Node("Job 2"), anchorNode).Attr("color", "brown")
	legend.Edge(legend.Node("Job 2"), legend.Node("Job 3")).Attr("color", "blue")
	legend.Edge(legend.Node("Job 2"), legend.Node("Task 1")).Attr("color", "blue")
	legend.Edge(legend.Node("Job 3"), legend.Node("Task 2")).Attr("color", "blue")
}
