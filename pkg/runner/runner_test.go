package runner

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/pkg/variables"

	taskpkg "github.com/taskctl/taskctl/pkg/task"
)

func TestTaskRunner_Run(t *testing.T) {
	c := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"true"}, []string{"false"}, []string{"echo 1"}, []string{"echo 2"})

	runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"local": c}))
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = ioutil.Discard, ioutil.Discard
	runner.WithVariable("Root", "/")

	task1 := taskpkg.NewTask()
	task1.Context = "local"
	task1.ExportAs = "EXPORT_NAME"

	task1.Commands = []string{"echo 'taskctl'"}
	task1.Name = "some test task"
	task1.Dir = "{{.Root}}"

	err = runner.Run(task1)
	if err != nil {
		t.Fatal(err)
	}

	if task1.Start.IsZero() || task1.End.IsZero() {
		t.Error()
	}

	if !strings.Contains(task1.Output(), "taskctl") {
		t.Error()
	}

	if task1.Errored || task1.ExitCode != 0 {
		t.Error()
	}

	if task1.Error != nil || task1.ErrorMessage() != "" {
		t.Error()
	}

	d := 1 * time.Minute
	task2 := taskpkg.NewTask()
	task2.Timeout = &d
	task2.Variations = []map[string]string{{"GOOS": "windows"}, {"GOOS": "linux"}}

	task2.Commands = []string{"false"}
	task2.Name = "some test task"
	task2.Dir = "{{.Root}}"

	err = runner.Run(task2)
	if err == nil {
		t.Fatal()
	}

	if !task2.Errored {
		t.Error()
	}

	task3 := taskpkg.NewTask()
	task3.Condition = "exit 1"
	err = runner.Run(task3)
	if err != nil || !task3.Skipped || task3.ExitCode != -1 {
		t.Error()
	}

	if task3.Duration() < 0 {
		t.Error()
	}

	runner.Finish()
}

func ExampleTaskRunner_Run() {
	t := taskpkg.FromCommands("go fmt ./...", "go build ./..")
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
