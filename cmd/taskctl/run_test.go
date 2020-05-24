package main

import "testing"

func Test_runCommand(t *testing.T) {
	tests := []appTest{
		{args: []string{"", "-c", "testdata/graph.yaml", "run", "graph:task2"}, errored: true},
		{args: []string{"", "-c", "testdata/graph.yaml", "run"}, errored: true},

		{args: []string{"", "-c", "testdata/graph.yaml", "run", "graph:task1"}, exactOutput: "hello, world!\n"},
		{args: []string{"", "-c", "testdata/graph.yaml", "run", "task", "graph:task1"}, exactOutput: "hello, world!\n"},
		{args: []string{"", "-c", "testdata/graph.yaml", "run", "pipeline", "graph:pipeline1"}, output: []string{"graph:task3", "hello, world!\n"}},
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
