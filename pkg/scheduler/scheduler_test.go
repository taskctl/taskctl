package scheduler_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/runner"

	"github.com/Ensono/taskctl/pkg/task"
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
	stage1 := &scheduler.Stage{
		Name: "stage1",
		Task: task.FromCommands("t1", "/usr/bin/true"),
	}
	stage2 := &scheduler.Stage{
		Name:      "stage2",
		Task:      task.FromCommands("t2", "/usr/bin/false"),
		DependsOn: []string{"stage1"},
	}
	stage3 := &scheduler.Stage{
		Name:      "stage3",
		Task:      task.FromCommands("t2", "/usr/bin/false"),
		DependsOn: []string{"stage2"},
	}
	stage4 := &scheduler.Stage{
		Name:      "stage4",
		Task:      task.FromCommands("t3", "true"),
		DependsOn: []string{"stage3"},
	}

	graph, err := scheduler.NewExecutionGraph(stage1, stage2, stage3, stage4)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := scheduler.NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err == nil {
		t.Fatal(err)
	}

	if graph.Duration() <= 0 {
		t.Fatal()
	}

	if stage3.Status != scheduler.StatusCanceled || stage4.Status != scheduler.StatusCanceled {
		t.Fatal("stage3 was not cancelled")
	}
}

func TestExecutionGraph_Scheduler_AllowFailure(t *testing.T) {
	stage1 := &scheduler.Stage{
		Name: "stage1",
		Task: task.FromCommands("t1", "true"),
	}
	stage2 := &scheduler.Stage{
		Name:         "stage2",
		Task:         task.FromCommands("t2", "false"),
		AllowFailure: true,
		DependsOn:    []string{"stage1"},
	}
	stage3 := &scheduler.Stage{
		Name:      "stage3",
		Task:      task.FromCommands("t3", "{{.command}}"),
		DependsOn: []string{"stage2"},
		Variables: variables.FromMap(map[string]string{"command": "true"}),
		Env:       variables.NewVariables(),
	}

	graph, err := scheduler.NewExecutionGraph(stage1, stage2, stage3)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := scheduler.NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if stage3.Status == scheduler.StatusCanceled {
		t.Fatal("stage3 was cancelled")
	}

	if stage3.Duration() <= 0 {
		t.Error()
	}

	schdlr.Finish()
}

func TestSkippedStage(t *testing.T) {
	stage1 := &scheduler.Stage{
		Name:      "stage1",
		Task:      task.FromCommands("t1", "true"),
		Condition: "true",
	}
	stage2 := &scheduler.Stage{
		Name:         "stage2",
		Task:         task.FromCommands("t2", "false"),
		AllowFailure: true,
		DependsOn:    []string{"stage1"},
		Condition:    "false",
	}

	graph, err := scheduler.NewExecutionGraph(stage1, stage2)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := scheduler.NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if stage1.Status != scheduler.StatusDone || stage2.Status != scheduler.StatusSkipped {
		t.Error()
	}
}

func TestScheduler_Cancel(t *testing.T) {
	stage1 := &scheduler.Stage{
		Name: "stage1",
		Task: task.FromCommands("t1", "sleep 60"),
	}

	graph, err := scheduler.NewExecutionGraph(stage1)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := scheduler.NewScheduler(taskRunner)
	go func() {
		schdlr.Cancel()
	}()

	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if schdlr.Cancelled() != 1 {
		t.Error()
	}
}

func TestConditionErroredStage(t *testing.T) {
	stage1 := &scheduler.Stage{
		Name:      "stage1",
		Task:      task.FromCommands("t1", "true"),
		Condition: "true",
	}
	stage2 := &scheduler.Stage{
		Name:         "stage2",
		Task:         task.FromCommands("t2", "false"),
		AllowFailure: true,
		DependsOn:    []string{"stage1"},
		Condition:    "/unknown-bin",
	}

	graph, err := scheduler.NewExecutionGraph(stage1, stage2)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := scheduler.NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if stage1.Status != scheduler.StatusDone || stage2.Status != scheduler.StatusError {
		t.Error()
	}
}

func ExampleScheduler_Schedule() {
	format := task.FromCommands("t1", "go fmt ./...")
	build := task.FromCommands("t2", "go build ./..")
	r, _ := runner.NewTaskRunner()
	s := scheduler.NewScheduler(r)

	graph, err := scheduler.NewExecutionGraph(
		&scheduler.Stage{Name: "format", Task: format},
		&scheduler.Stage{Name: "build", Task: build, DependsOn: []string{"format"}},
	)
	if err != nil {
		return
	}

	err = s.Schedule(graph)
	if err != nil {
		fmt.Println(err)
	}
}
