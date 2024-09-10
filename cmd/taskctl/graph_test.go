package cmd_test

import (
	"testing"
)

func Test_graphCommand(t *testing.T) {

	t.Run("errors with pipeline missing", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:    []string{"-c", "testdata/graph.yaml", "graph"},
			errored: true,
		})
	})
	t.Run("succeeds with pipeline specified", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:   []string{"-c", "testdata/graph.yaml", "graph", "graph:pipeline1"},
			output: []string{"label=\"graph:pipeline2\"", "label=\"graph:task1\""},
		})
	})

	t.Run("succeeds with pipeline specified left to right", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:   []string{"-c", "testdata/graph.yaml", "graph", "--lr", "graph:pipeline1"},
			output: []string{"rankdir=\"LR\""},
		})
	})
}
