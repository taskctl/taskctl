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

	err = g.AddStage(&scheduler.Stage{Name: "stage1", DependsOn: []string{"stage2"}})
	if err != nil {
		t.Fatal()
	}
	err = g.AddStage(&scheduler.Stage{Name: "stage2", DependsOn: []string{"stage1"}})
	if err == nil {
		t.Fatal("add stage cycle detection failed")
	}
}
