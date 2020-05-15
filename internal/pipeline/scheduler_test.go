package pipeline

import (
	"errors"
	"testing"

	"github.com/taskctl/taskctl/internal/task"
)

type TestTaskRunner struct {
}

func (t2 TestTaskRunner) Run(t *task.Task) error {
	if t.Commands[0] == "/usr/bin/false" {
		t.ExitCode = 1
		t.Errored = true
		return errors.New("task failed")
	}

	return nil
}

func (t2 TestTaskRunner) Cancel() {}

func (t2 TestTaskRunner) Finish() {}

func TestExecutionGraph_Scheduler(t *testing.T) {
	graph := NewExecutionGraph()

	stage1 := &Stage{
		Name: "stage1",
		Task: task.FromCommand("/usr/bin/true"),
	}
	stage2 := &Stage{
		Name:      "stage2",
		Task:      task.FromCommand("/usr/bin/false"),
		DependsOn: []string{"stage1"},
	}
	stage3 := &Stage{
		Name:      "stage3",
		Task:      task.FromCommand("/usr/bin/false"),
		DependsOn: []string{"stage2"},
	}

	graph.AddStage(stage1)
	graph.AddStage(stage2)
	graph.AddStage(stage3)

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	err := schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if schdlr.Start.IsZero() || schdlr.End.IsZero() {
		t.Fatal()
	}

	if stage3.Status != StatusCanceled {
		t.Fatal("stage3 was not cancelled")
	}
}
