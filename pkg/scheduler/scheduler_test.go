package scheduler

import (
	"errors"
	"fmt"
	"testing"

	"github.com/taskctl/taskctl/pkg/runner"

	"github.com/taskctl/taskctl/pkg/task"
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
	graph, _ := NewExecutionGraph()

	stage1 := &Stage{
		Name: "stage1",
		Task: task.FromCommands("/usr/bin/true"),
	}
	stage2 := &Stage{
		Name:      "stage2",
		Task:      task.FromCommands("/usr/bin/false"),
		DependsOn: []string{"stage1"},
	}
	stage3 := &Stage{
		Name:      "stage3",
		Task:      task.FromCommands("/usr/bin/false"),
		DependsOn: []string{"stage2"},
	}

	err := graph.AddStage(stage1)
	if err != nil {
		t.Fatal(err)
	}
	err = graph.AddStage(stage2)
	if err != nil {
		t.Fatal(err)
	}
	err = graph.AddStage(stage3)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err == nil {
		t.Fatal(err)
	}

	if graph.start.IsZero() || graph.end.IsZero() {
		t.Fatal()
	}

	if stage3.Status != StatusCanceled {
		t.Fatal("stage3 was not cancelled")
	}
}

func TestExecutionGraph_Scheduler_AllowFailure(t *testing.T) {
	stage1 := &Stage{
		Name: "stage1",
		Task: task.FromCommands("true"),
	}
	stage2 := &Stage{
		Name:         "stage2",
		Task:         task.FromCommands("false"),
		AllowFailure: true,
		DependsOn:    []string{"stage1"},
	}
	stage3 := &Stage{
		Name:      "stage3",
		Task:      task.FromCommands("true"),
		DependsOn: []string{"stage2"},
	}

	graph, err := NewExecutionGraph(stage1, stage2, stage3)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if stage3.Status == StatusCanceled {
		t.Fatal("stage3 was cancelled")
	}
}

func ExampleScheduler_Schedule() {
	format := task.FromCommands("go fmt ./...")
	build := task.FromCommands("go build ./..")
	r, _ := runner.NewTaskRunner()
	s := NewScheduler(r)

	graph, err := NewExecutionGraph(
		&Stage{Name: "format", Task: format},
		&Stage{Name: "build", Task: build, DependsOn: []string{"format"}},
	)
	if err != nil {
		return
	}

	err = s.Schedule(graph)
	if err != nil {
		fmt.Println(err)
	}
}
