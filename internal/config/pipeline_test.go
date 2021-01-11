package config

import (
	"github.com/taskctl/taskctl/pkg/scheduler"
	"strings"
	"testing"
)

func TestBuildPipeline_Cyclic(t *testing.T) {
	cfg := NewConfig()

	stages := []*stageDefinition{
		{
			Name:      "task1",
			Task:      "task1",
			DependsOn: []string{"last-stage"},
			Dir:       "/root",
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

	g, _ := scheduler.NewExecutionGraph()
	_, err = buildPipeline(g, stages, cfg)
	if err == nil || err.Error() != "cycle detected" {
		t.Errorf("cycles detection failed")
	}
}

func TestBuildPipeline_Error(t *testing.T) {
	cfg := NewConfig()

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

	stages1 := []*stageDefinition{
		{
			Name:      "task4",
			Task:      "task4",
			DependsOn: []string{"last-stage"},
			Dir:       "/root",
		},
	}

	g, _ := scheduler.NewExecutionGraph()
	_, err = buildPipeline(g, stages1, cfg)
	if err == nil || !strings.Contains(err.Error(), "no such task") {
		t.Error()
	}

	stages2 := []*stageDefinition{
		{
			Name:      "task1",
			Pipeline:  "pipeline1",
			DependsOn: []string{"last-stage"},
			Dir:       "/root",
		},
	}

	g, _ = scheduler.NewExecutionGraph()
	_, err = buildPipeline(g, stages2, cfg)
	if err == nil || !strings.Contains(err.Error(), "no such pipeline") {
		t.Error()
	}

	stages3 := []*stageDefinition{
		{
			Name:      "task1",
			Task:      "task1",
			DependsOn: []string{"last-stage"},
			Dir:       "/root",
		},
		{
			Name:      "task1",
			Task:      "task1",
			DependsOn: []string{"last-stage"},
			Dir:       "/root",
		},
	}

	g, _ = scheduler.NewExecutionGraph()
	_, err = buildPipeline(g, stages3, cfg)
	if err == nil || !strings.Contains(err.Error(), "stage with same name") {
		t.Error()
	}
}
