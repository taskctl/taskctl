package config

import (
	"testing"

	"github.com/taskctl/taskctl/pkg/task"
)

func Test_buildWatcher(t *testing.T) {
	_, err := buildWatcher("tw", &watcherDefinition{
		Task: "hello",
	}, &Config{})
	if err == nil {
		t.Error()
	}

	_, err = buildWatcher("tw", &watcherDefinition{
		Task: "hello",
	}, &Config{Tasks: map[string]*task.Task{"hello": {}}})
	if err != nil {
		t.Fatal()
	}
}
