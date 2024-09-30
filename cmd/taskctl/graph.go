package cmd

import (
	"fmt"
	"io"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/emicklei/dot"
	"github.com/spf13/cobra"
)

type graphFlags struct {
	leftToRight bool
}

type graphCmd struct {
	channelOut, channelErr io.Writer
}

func newGraphCmd(rootCmd *TaskCtlCmd) {
	f := &graphFlags{}
	gc := &graphCmd{
		channelOut: rootCmd.ChannelOut,
		channelErr: rootCmd.ChannelErr,
	}
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
			pipelineName := args[0]
			return gc.graphCmdRun(pipelineName, conf)
		},
	}

	graphCmd.PersistentFlags().BoolVarP(&f.leftToRight, "lr", "", false, "orients outputted graph left-to-right")
	_ = rootCmd.viperConf.BindPFlag("lr", graphCmd.PersistentFlags().Lookup("lr"))

	rootCmd.Cmd.AddCommand(graphCmd)
}

func (gc *graphCmd) graphCmdRun(name string, conf *config.Config) error {

	p := conf.Pipelines[name]
	if p == nil {
		return fmt.Errorf("no such pipeline %s", name)
	}

	g := dot.NewGraph(dot.Directed)
	g.Attr("center", "true")
	isLr := conf.Options.GraphOrientationLeftRight
	if isLr {
		g.Attr("rankdir", "LR")
	}

	draw(g, p)

	fmt.Fprintln(gc.channelOut, g.String())

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
