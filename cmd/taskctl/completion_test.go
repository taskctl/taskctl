package main

import "testing"

func Test_completionCommand(t *testing.T) {
	app := makeTestApp(t)

	tests := []appTest{
		{args: []string{"", "completion"}},
		{args: []string{"", "completion", "--help"}},
		{args: []string{"", "completion", "bash"}},
		{args: []string{"", "completion", "zsh"}},
		{args: []string{"", "completion", "unknown-shell"}, errored: true},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
	}
}
