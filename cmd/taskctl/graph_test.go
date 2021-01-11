package main

import (
	"testing"
)

func Test_graphCommand(t *testing.T) {
	app := makeTestApp(t)

	tests := []appTest{
		{
			args:    []string{"", "-c", "testdata/graph.yaml", "graph"},
			errored: true,
		},
		{
			args:   []string{"", "-c", "testdata/graph.yaml", "graph", "graph:pipeline1"},
			output: []string{"label=\"graph:pipeline2\"", "label=\"graph:task1\""},
		},
		{
			args:   []string{"", "-c", "testdata/graph.yaml", "graph", "--lr", "graph:pipeline1"},
			output: []string{"rankdir=\"LR\""},
		},
	}

	for _, test := range tests {
		runAppTest(app, test, t)
	}
}
