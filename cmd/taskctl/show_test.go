package cmd_test

import "testing"

func Test_showCommand(t *testing.T) {
	t.Run("errors on args", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:    []string{"-c", "testdata/graph.yaml", "show"},
			errored: true,
		})
	})
	t.Run("errors on incorrect task name", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:    []string{"-c", "testdata/graph.yaml", "show", "task:unknown"},
			errored: true,
		})
	})
	t.Run("succeds on args", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:   []string{"-c", "testdata/graph.yaml", "show", "graph:task1"},
			output: []string{"Name: graph:task1", "echo &#39;hello, world!&#39"},
		})
	})
}
