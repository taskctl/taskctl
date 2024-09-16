package cmd_test

import (
	"os"
	"testing"
)

func Test_showCommand(t *testing.T) {
	t.Run("errors on args", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/graph.yaml", "show"},
			errored: true,
		})
	})
	t.Run("errors on incorrect task name", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/graph.yaml", "show", "task:unknown"},
			errored: true,
		})
	})
	t.Run("succeeds on args", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/graph.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:   []string{"-c", "testdata/graph.yaml", "show", "graph:task1"},
			output: []string{"Name: graph:task1", "echo &#39;hello, world!&#39"},
		})
	})
}
