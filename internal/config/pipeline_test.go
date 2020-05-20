package config

import (
	"testing"
)

func TestBuildPipeline_Cyclic(t *testing.T) {
	cfg := NewConfig()

	stages := []*stageDefinition{
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

	tasks := map[string]*taskDefinition{
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

	var err error
	for k, v := range tasks {
		cfg.Tasks[k], err = buildTask(v)
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = buildPipeline(stages, cfg)
	if err == nil || err.Error() != "cycle detected" {
		t.Errorf("cycles detection failed")
	}
}
