package runner

import (
	"fmt"
	"testing"

	"github.com/taskctl/taskctl/pkg/task"
)

func TestTaskRunner_Run(t *testing.T) {
	task := task.NewTask()

	task.Commands = []string{"true"}
	task.Name = "some test task"
	task.Dir = "{{.Root}}"

	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}

	runner.WithVariable("Root", "/")

	runner.DryRun = true

	err = runner.Run(task)
	if err != nil {
		t.Fatal(err)
	}

	if task.Start.IsZero() || task.End.IsZero() {
		t.Fatal()
	}

	if task.Errored || task.ExitCode != 0 {
		t.Fatal()
	}
}

func ExampleTaskRunner_Run() {
	t := task.FromCommands("go fmt ./...", "go build ./..")
	r, err := NewTaskRunner()
	if err != nil {
		return
	}
	err = r.Run(t)
	if err != nil {
		fmt.Println(err, t.ExitCode, t.ErrorMessage())
	}
	fmt.Println(t.Output())
}
