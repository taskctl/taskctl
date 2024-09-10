package cmd_test

import (
	"testing"
)

func Test_runCommand(t *testing.T) {
	t.Run("errors on graph:task4", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "graph:task4"}, errored: true})
	})
	t.Run("no task/pipeline supplied", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "graph:task4"}, errored: true})
	})
	t.Run("correct output assigned without specifying task or pipeline", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "graph:task1"}, exactOutput: "hello, world!\n"})
	})
	t.Run("correct with task specified", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "task", "graph:task1"}, exactOutput: "hello, world!\n"})
	})
	t.Run("correct with pipeline specified", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "pipeline", "graph:pipeline1"}, output: []string{"hello, world!\n"}})
	})
	t.Run("correct prefixed output", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--output=prefixed", "-c", "testdata/graph.yaml", "run", "graph:pipeline1"}, output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"}})
	})
}

func Test_runCommandWithArgumentsList(t *testing.T) {
	t.Run("with args - first arg", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/task.yaml", "run", "task", "task:task1", "--", "first", "second"}, exactOutput: "This is first argument\n"})
	})
	t.Run("with args - second arg", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/task.yaml", "run", "task", "task:task2", "--", "first", "second"}, exactOutput: "This is second argument\n"})
	})
	t.Run("with argsList - - first and second arg", func(t *testing.T) {
		runTestHelper(t, runTestIn{args: []string{"--raw", "-c", "testdata/task.yaml", "run", "task", "task:task3", "--", "first", "and", "second"}, exactOutput: "This is first and second arguments\n"})
	})
}
