package cmd

import (
	"fmt"
	"maps"
	"slices"

	"github.com/emicklei/dot"
	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/scheduler"
)

func newGraphCommand(cfg *config.Config) *cobra.Command {
	var lr bool

	graphCmd := &cobra.Command{
		Use:     "graph PIPELINE",
		Aliases: []string{"g"},
		Short:   "visualizes pipeline execution graph",
		Long: "Generates a visual representation of pipeline execution plan. " +
			"The output is in the DOT format, which can be used by GraphViz to generate charts.",
		Example:           "  taskctl graph pipeline1 | dot -Tsvg > graph.svg",
		GroupID:           groupInspect,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: pipelineCompletion(cfg),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			p := cfg.Pipelines[name]
			if p == nil {
				return fmt.Errorf("no such pipeline %s", name)
			}

			g := dot.NewGraph(dot.Directed)
			g.Attr("center", "true")
			if lr {
				g.Attr("rankdir", "LR")
			}

			draw(g, p)
			fmt.Println(g.String())

			return nil
		},
	}

	graphCmd.Flags().BoolVar(&lr, "lr", false, "orients the output graph left-to-right")

	return graphCmd
}

// pipelineCompletion completes pipeline names only; graph rejects tasks.
func pipelineCompletion(cfg *config.Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completionFunc(cfg, func() []string {
		return slices.Sorted(maps.Keys(cfg.Pipelines))
	})
}

func draw(g *dot.Graph, p *scheduler.ExecutionGraph) {
	for k, v := range p.Nodes() {
		if v.Pipeline != nil {
			cluster := g.Subgraph(k, dot.ClusterOption{})
			draw(cluster, v.Pipeline)
		}

		for _, from := range p.To(k) {
			g.Edge(g.Node(from), g.Node(k))
		}
	}
}
