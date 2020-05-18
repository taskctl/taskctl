package runner

import (
	"testing"

	taskpkg "github.com/taskctl/taskctl/pkg/task"
)

func TestTaskRunner_Run(t *testing.T) {
	task := taskpkg.NewTask()

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
