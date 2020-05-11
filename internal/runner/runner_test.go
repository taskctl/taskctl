package runner

import (
	"testing"

	"github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/output"
	taskpkg "github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/utils"
)

func TestTaskRunner_Run(t *testing.T) {
	task := taskpkg.NewTask()

	task.Command = []string{"some-command-to-run"}
	task.Context = "local"
	task.Name = "some test task"
	task.Dir = "{{.Root}}"

	runner, err := NewTaskRunner(
		map[string]*context.ExecutionContext{"local": &context.ExecutionContext{}},
		output.OutputFormatRaw,
		utils.NewVariables(map[string]string{"Root": "/tmp"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	runner.DryRun()

	err = runner.Run(task, utils.NewVariables(nil), utils.NewVariables(nil))
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
