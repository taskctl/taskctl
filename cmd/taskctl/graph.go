package main

import (
	"errors"
	"fmt"

	"github.com/emicklei/dot"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/pkg/scheduler"
)

func newGraphCommand() *cli.Command {
	return &cli.Command{
		Name:      "graph",
		Aliases:   []string{"g"},
		Usage:     "visualizes pipeline execution graph",
		UsageText: "taskctl graph [pipeline] | dot -Tsvg > graph.svg",
		Description: "Generates a visual representation of pipeline execution plan. " +
			"The output is in the DOT format, which can be used by GraphViz to generate charts.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "lr",
				Usage: "orients outputted graph left-to-right",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				err := cli.ShowCommandHelp(c, "graph")
				if err != nil {
					logrus.Error(err)
				}
				return errors.New("no pipeline set")
			}
			name := c.Args().First()
			p := cfg.Pipelines[name]
			if p == nil {
				return fmt.Errorf("no such pipeline %s", name)
			}

			g := dot.NewGraph(dot.Directed)
			g.Attr("center", "true")
			if c.Bool("lr") {
				g.Attr("rankdir", "LR")
			}

			draw(g, p)

			fmt.Println(g.String())

			return nil
		},
	}
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
