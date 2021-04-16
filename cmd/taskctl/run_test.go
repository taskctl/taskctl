package main

import "testing"

func Test_runCommand(t *testing.T) {
	tests := []appTest{
		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "run", "graph:task2"}, errored: true},
		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "run"}, errored: true},
		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "run", "graph:task1"}, exactOutput: "hello, world!\n"},
		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "run", "task", "graph:task1"}, exactOutput: "hello, world!\n"},
		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "run", "pipeline", "graph:pipeline1"}, output: []string{"graph:task3", "hello, world!\n"}},
		{
			args:   []string{"", "--output=prefixed", "-c", "testdata/graph.yaml", "run", "graph:pipeline1"},
			output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"},
		},
	}

	for _, v := range tests {
		app := makeTestApp(t)
		runAppTest(app, v, t)
	}
}

func Test_runCommandWithArgumentsList(t *testing.T) {
	tests := []appTest{
		{args: []string{"", "--raw", "-c", "testdata/task.yaml", "run", "task", "task:task1", "--", "first", "second"}, exactOutput: "This is first argument\n"},
		{args: []string{"", "--raw", "-c", "testdata/task.yaml", "run", "task", "task:task2", "--", "first", "second"}, exactOutput: "This is second argument\n"},
		{args: []string{"", "--raw", "-c", "testdata/task.yaml", "run", "task", "task:task3", "--", "first", "and", "second"}, exactOutput: "This is first and second arguments\n"},
	}

	for _, v := range tests {
		app := makeTestApp(t)
		runAppTest(app, v, t)
	}
}
