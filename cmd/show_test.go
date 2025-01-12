package cmd_test

import "testing"

func Test_showCommand(t *testing.T) {
	app := makeTestApp()

	tests := []appTest{
		{args: []string{"", "-c", "testdata/graph.yaml", "show"}, errored: true},
		{args: []string{"", "-c", "testdata/graph.yaml", "show", "graph:task1"}, output: []string{"Name: graph:task1", "echo 'hello, world!'"}},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
	}
}
