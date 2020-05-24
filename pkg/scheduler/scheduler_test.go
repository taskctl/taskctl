package scheduler

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/taskctl/taskctl/pkg/variables"

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
	stage4 := &Stage{
		Name:      "stage4",
		Task:      task.FromCommands("true"),
		DependsOn: []string{"stage3"},
	}

	graph, err := NewExecutionGraph(stage1, stage2, stage3, stage4)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err == nil {
		t.Fatal(err)
	}

	if graph.Duration() <= 0 {
		t.Fatal()
	}

	if stage3.Status != StatusCanceled || stage4.Status != StatusCanceled {
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
		Task:      task.FromCommands("{{.command}}"),
		DependsOn: []string{"stage2"},
		Variables: variables.FromMap(map[string]string{"command": "true"}),
		Env:       variables.NewVariables(),
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

	if stage3.Duration() <= 0 {
		t.Error()
	}

	schdlr.Finish()
}

func TestSkippedStage(t *testing.T) {
	stage1 := &Stage{
		Name:      "stage1",
		Task:      task.FromCommands("true"),
		Condition: "true",
	}
	stage2 := &Stage{
		Name:         "stage2",
		Task:         task.FromCommands("false"),
		AllowFailure: true,
		DependsOn:    []string{"stage1"},
		Condition:    "false",
	}

	graph, err := NewExecutionGraph(stage1, stage2)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if stage1.Status != StatusDone || stage2.Status != StatusSkipped {
		t.Error()
	}
}

func TestScheduler_Cancel(t *testing.T) {
	stage1 := &Stage{
		Name: "stage1",
		Task: task.FromCommands("sleep 60"),
	}

	graph, err := NewExecutionGraph(stage1)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	go func() {
		schdlr.Cancel()
	}()

	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&schdlr.cancelled) != 1 {
		t.Error()
	}
}

func TestConditionErroredStage(t *testing.T) {
	stage1 := &Stage{
		Name:      "stage1",
		Task:      task.FromCommands("true"),
		Condition: "true",
	}
	stage2 := &Stage{
		Name:         "stage2",
		Task:         task.FromCommands("false"),
		AllowFailure: true,
		DependsOn:    []string{"stage1"},
		Condition:    "/unknown-bin",
	}

	graph, err := NewExecutionGraph(stage1, stage2)
	if err != nil {
		t.Fatal(err)
	}

	taskRunner := TestTaskRunner{}

	schdlr := NewScheduler(taskRunner)
	err = schdlr.Schedule(graph)
	if err != nil {
		t.Fatal(err)
	}

	if stage1.Status != StatusDone || stage2.Status != StatusError {
		t.Error()
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
