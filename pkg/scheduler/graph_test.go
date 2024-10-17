package scheduler_test

import (
	"testing"

	"github.com/Ensono/taskctl/pkg/scheduler"
)

func TestExecutionGraph_AddStage(t *testing.T) {
	g, err := scheduler.NewExecutionGraph()
	if err != nil {
		t.Fatal(err)
	}

	err = g.AddStage(scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage1"
		s.DependsOn = []string{"stage2"}
	}))
	if err != nil {
		t.Fatal()
	}
	err = g.AddStage(scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage2"
		s.DependsOn = []string{"stage1"}
	}))
	if err == nil {
		t.Fatal("add stage cycle detection failed")
	}
}
