package scheduler

import "testing"

func TestExecutionGraph_AddStage(t *testing.T) {
	g, err := NewExecutionGraph()
	if err != nil {
		t.Fatal(err)
	}

	err = g.AddStage(&Stage{Name: "stage1", DependsOn: []string{"stage2"}})
	if err != nil {
		t.Fatal()
	}
	err = g.AddStage(&Stage{Name: "stage2", DependsOn: []string{"stage1"}})
	if err == nil {
		t.Fatal("add stage cycle detection failed")
	}
}
