package cmd_test

import (
	"testing"
)

func Test_graphCommand(t *testing.T) {

	tests := []appTest{
		{
			args:    []string{"-c", "testdata/graph.yaml", "graph"},
			errored: true,
		},
		{
			args:   []string{"-c", "testdata/graph.yaml", "graph", "graph:pipeline1"},
			output: []string{"label=\"graph:pipeline2\"", "label=\"graph:task1\""},
		},
		{
			args:   []string{"-c", "testdata/graph.yaml", "graph", "--lr", "graph:pipeline1"},
			output: []string{"rankdir=\"LR\""},
		},
		// completion offers only pipelines, never tasks.
		{
			args:   []string{"__complete", "-c", "testdata/graph.yaml", "graph", ""},
			output: []string{"graph:pipeline1"}, absent: []string{"graph:task1"},
		},
	}

	for _, test := range tests {
		runAppTest(t, test)
	}
}
