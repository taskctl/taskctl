package cmd

import (
	"fmt"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/emicklei/dot"
	"github.com/spf13/cobra"
)

var (
	leftToRight bool
	graphCmd    = &cobra.Command{
		Use:     "graph",
		Aliases: []string{"g"},
		Short:   `visualizes pipeline execution graph`,
		Long: `Generates a visual representation of pipeline execution plan.
The output is in the DOT format, which can be used by GraphViz to generate charts.`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return buildTaskRunner(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return graphCmdRun(args[0])
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
)

func init() {
	graphCmd.PersistentFlags().BoolVarP(&leftToRight, "lr", "", false, "orients outputted graph left-to-right")

	TaskCtlCmd.AddCommand(graphCmd)
}

func graphCmdRun(name string) error {

	p := conf.Pipelines[name]
	if p == nil {
		return fmt.Errorf("no such pipeline %s", name)
	}

	g := dot.NewGraph(dot.Directed)
	g.Attr("center", "true")
	if leftToRight {
		g.Attr("rankdir", "LR")
	}

	draw(g, p)

	fmt.Fprintln(ChannelOut, g.String())

	return nil
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
