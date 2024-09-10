package cmd_test

import "testing"

func Test_validateCommand(t *testing.T) {

	t.Run("errors with missing config", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:    []string{"validate", "testdata/graph2.yaml"},
			errored: true,
		})
	})

	t.Run("succeeds with correct config", func(t *testing.T) {
		runTestHelper(t, runTestIn{
			args:   []string{"validate", "testdata/graph.yaml"},
			output: []string{"file is valid"},
		})
	})
}
