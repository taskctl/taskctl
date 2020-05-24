package main

import "testing"

func Test_listCommand(t *testing.T) {
	app := makeTestApp(t)

	tests := []appTest{
		{args: []string{"", "-c", "testdata/graph.yaml", "list"}, output: []string{"graph:pipeline1", "graph:task1", "no watchers"}},
		{args: []string{"", "-c", "testdata/graph.yaml", "list", "pipelines"}, output: []string{"graph:pipeline1"}},
		{args: []string{"", "-c", "testdata/graph.yaml", "list", "tasks"}, output: []string{"graph:task1"}},
		{args: []string{"", "-c", "testdata/graph.yaml", "list", "watchers"}, exactOutput: ""},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
	}
}
