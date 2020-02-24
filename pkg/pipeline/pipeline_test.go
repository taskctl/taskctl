package pipeline

import (
	"testing"

	"github.com/taskctl/taskctl/pkg/builder"
)

func TestBuildPipeline_Cyclic(t *testing.T) {
	stages := []*builder.StageDefinition{
		{
			Name:      "task1",
			Task:      "task1",
			DependsOn: []string{"last-stage"},
		},
		{
			Name:      "task2",
			Task:      "task2",
			DependsOn: []string{"task1"},
			Env:       nil,
		},
		{
			Name:      "last-stage",
			Task:      "task3",
			DependsOn: []string{"task2"},
		},
	}

	tasks := map[string]*builder.TaskDefinition{
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

	_, err := BuildPipeline(stages, make(map[string][]*builder.StageDefinition), tasks)
	if err == nil || err.Error() != "cycle detected" {
		t.Errorf("cycles detection failed")
	}
}
