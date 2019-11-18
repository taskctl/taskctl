package scheduler

import (
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/task"
	"testing"
)

func TestBuildPipeline_Cyclic(t *testing.T) {
	stages := []config.Stage{
		{
			Task:      "task1",
			DependsOn: []string{"task3"},
		},
		{
			Task:      "task2",
			DependsOn: []string{"task1"},
			Env:       nil,
		},
		{
			Task:      "task3",
			DependsOn: []string{"task2"},
		},
	}

	tasks := map[string]*task.Task{
		"task1": {
			Name: "task1",
		},
		"task2": {
			Name: "task2",
		},
		"task3": {
			Name: "task3",
		},
	}

	_, err := BuildPipeline(stages, tasks)
	if err == nil || err.Error() != "cycle detected" {
		t.Errorf("cycles detection failed")
	}
}
