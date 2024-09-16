package cmd_test

import (
	"os"
	"testing"
)

func Test_validateCommand(t *testing.T) {

	t.Run("errors with missing config", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"validate", "testdata/graph2.yaml"},
			errored: true,
		})
	})

	t.Run("succeeds with correct config", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/graph.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:   []string{"validate", "testdata/graph.yaml"},
			output: []string{"file is valid"},
		})
	})
}
