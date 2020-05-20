package config

import (
	"github.com/taskctl/taskctl/pkg/task"
	"testing"
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
