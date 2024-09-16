package cmd_test

import (
	"os"
	"testing"
)

func Test_graphCommand(t *testing.T) {

	t.Run("errors with pipeline missing", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/graph.yaml", "graph"},
			errored: true,
		})
	})

	t.Run("succeeds with pipeline specified", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:   []string{"-c", "testdata/graph.yaml", "graph", "graph:pipeline1"},
			output: []string{"label=\"graph:pipeline2\"", "label=\"graph:task1\""},
		})
	})

	t.Run("succeeds with pipeline specified left to right", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/graph.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:   []string{"graph", "--lr", "graph:pipeline1", "-d", "--dry-run"},
			output: []string{"rankdir=\"LR\""},
		})
	})
}
