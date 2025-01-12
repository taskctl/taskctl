package cmd

import "testing"

func Test_validateCommand(t *testing.T) {
	app := makeTestApp()

	tests := []appTest{
		{args: []string{"", "validate", "testdata/graph2.yaml"}, errored: true},
		{args: []string{"", "validate", "testdata/graph.yaml"}, output: []string{"file is valid"}},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
	}
}
