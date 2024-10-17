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
	stage1 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage1"
		s.Task = task.FromCommands("t1", "/usr/bin/true")
	})
	stage2 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage2"
		s.Task = task.FromCommands("t2", "/usr/bin/false")
		s.DependsOn = []string{"stage1"}
	})

	stage3 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage3"
		s.Task = task.FromCommands("t2", "/usr/bin/false")
		s.DependsOn = []string{"stage2"}
	})

	stage4 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage4"
		s.Task = task.FromCommands("t3", "true")
		s.DependsOn = []string{"stage3"}

	})

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

	if stage3.Status.Load() != scheduler.StatusCanceled || stage4.Status.Load() != scheduler.StatusCanceled {
		t.Fatal("stage3 was not cancelled")
	}
}

func TestExecutionGraph_Scheduler_AllowFailure(t *testing.T) {
	stage1 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage1"
		s.Task = task.FromCommands("t1", "true")

	})
	stage2 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage2"
		s.Task = task.FromCommands("t2", "false")
		s.AllowFailure = true
		s.DependsOn = []string{"stage1"}

	})
	stage3 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage3"
		s.Task = task.FromCommands("t3", "{{.command}}")
		s.DependsOn = []string{"stage2"}
		s.Variables = variables.FromMap(map[string]string{"command": "true"})
		s.Env = variables.NewVariables()
	})

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

	if stage3.Status.Load() == scheduler.StatusCanceled {
		t.Fatal("stage3 was cancelled")
	}

	if stage3.Duration() <= 0 {
		t.Error()
	}

	schdlr.Finish()
}

func TestSkippedStage(t *testing.T) {
	stage1 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage1"
		s.Task = task.FromCommands("t1", "true")
		s.Condition = "true"

	})
	stage2 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage2"
		s.Task = task.FromCommands("t2", "false")
		s.AllowFailure = true
		s.DependsOn = []string{"stage1"}
		s.Condition = "false"
	})

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

	if stage1.Status.Load() != scheduler.StatusDone || stage2.Status.Load() != scheduler.StatusSkipped {
		t.Error()
	}
}

func TestScheduler_Cancel(t *testing.T) {
	stage1 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage1"
		s.Task = task.FromCommands("t1", "sleep 60")

	})

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
	stage1 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage1"
		s.Task = task.FromCommands("t1", "true")
		s.Condition = "true"
	})

	stage2 := scheduler.NewStage(func(s *scheduler.Stage) {
		s.Name = "stage2"
		s.Task = task.FromCommands("t2", "false")
		s.AllowFailure = true
		s.DependsOn = []string{"stage1"}
		s.Condition = "/unknown-bin"
	})

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

	if stage1.Status.Load() != scheduler.StatusDone || stage2.Status.Load() != scheduler.StatusError {
		t.Error()
	}
}

func ExampleScheduler_Schedule() {
	format := task.FromCommands("t1", "go fmt ./...")
	build := task.FromCommands("t2", "go build ./..")
	r, _ := runner.NewTaskRunner()
	s := scheduler.NewScheduler(r)

	graph, err := scheduler.NewExecutionGraph(
		scheduler.NewStage(func(s *scheduler.Stage) {
			s.Name = "format"
			s.Task = format
		}),
		scheduler.NewStage(func(s *scheduler.Stage) {
			s.Name = "build"
			s.Task = build
			s.DependsOn = []string{"format"}
		}),
	)
	if err != nil {
		return
	}

	err = s.Schedule(graph)
	if err != nil {
		fmt.Println(err)
	}
}
