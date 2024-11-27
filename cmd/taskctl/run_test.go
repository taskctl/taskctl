package cmd_test

import (
	"os"
	"testing"
)

func Test_runCommand(t *testing.T) {
	t.Run("errors on graph:task4", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/graph.yaml", "run", "graph:task4", "--raw"}, errored: true})
	})
	t.Run("no task or pipeline supplied", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/graph.yaml", "run", "graph:task4", "--raw"}, errored: true})
	})

	t.Run("correct output assigned without specifying task or pipeline", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/graph.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"run", "graph:task1", "--raw"}, exactOutput: "hello, world!\n"})
	})

	t.Run("correct with task specified", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/graph.yaml", "run", "task", "graph:task1", "--raw"}, exactOutput: "hello, world!\n"})
	})
	t.Run("correct with pipeline specified", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/graph.yaml", "run", "pipeline", "graph:pipeline1", "--raw"}, output: []string{"hello, world!\n"}})
	})
	t.Run("correct prefixed output", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/graph.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"--output=prefixed", "-c", "testdata/graph.yaml", "run", "graph:pipeline1"}, output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"}})
	})

	t.Run("correct with graph-only - denormalized", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/generate.yml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{
			args: []string{"run", "graph:pipeline1", "--graph-only"},
			output: []string{`[label="graph:pipeline1->dev_anchor",shape="point",style="invis"]`,
				`[label="graph:pipeline1->graph:pipeline3_anchor",shape="point",style="invis"]`, `label="graph:pipeline1->prod"`,
			},
		})
	})
}

func Test_runCommandWithArgumentsList(t *testing.T) {
	t.Run("with args - first arg", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/task.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/task.yaml", "run", "task", "task:task1", "--raw", "--", "first", "second"}, exactOutput: "This is first argument\n"})
	})
	t.Run("with args - second arg", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/task.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/task.yaml", "run", "task", "task:task2", "--raw", "--", "first", "second"}, exactOutput: "This is second argument\n"})
	})
	t.Run("with argsList - - first and second arg", func(t *testing.T) {
		os.Setenv("TASKCTL_CONFIG_FILE", "testdata/task.yaml")
		defer os.Unsetenv("TASKCTL_CONFIG_FILE")
		cmdRunTestHelper(t, &cmdRunTestInput{args: []string{"-c", "testdata/task.yaml", "run", "task", "task:task3", "--raw", "--", "first", "and", "second"}, exactOutput: "This is first and second arguments\n"})
	})
}

