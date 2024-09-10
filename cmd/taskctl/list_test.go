package cmd_test

import "testing"

func Test_listCommand(t *testing.T) {
	tests := map[string]runTestIn{
		"list all":       {args: []string{"-c", "testdata/graph.yaml", "list"}, output: []string{"graph:pipeline1", "graph:task1", "no watchers"}},
		"list pipelines": {args: []string{"-c", "testdata/graph.yaml", "list", "pipelines"}, output: []string{"graph:pipeline1"}},
		"list tasks":     {args: []string{"-c", "testdata/graph.yaml", "list", "tasks"}, output: []string{"graph:task1"}},
		"list watchers":  {args: []string{"-c", "testdata/graph.yaml", "list", "watchers"}, exactOutput: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			runTestHelper(t, tt)
		})
	}
}
