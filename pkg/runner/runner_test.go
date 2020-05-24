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

func TestTaskRunner(t *testing.T) {
	c := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"true"}, []string{"false"}, []string{"echo 1"}, []string{"echo 2"})

	runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"local": c}))
	if err != nil {
		t.Fatal(err)
	}
	runner.SetContexts(map[string]*ExecutionContext{
		"default": DefaultContext(),
		"local":   c,
	})
	if _, ok := runner.contexts["default"]; !ok {
		t.Error()
	}

	runner.Stdout, runner.Stderr = ioutil.Discard, ioutil.Discard
	runner.SetVariables(variables.FromMap(map[string]string{"Root": "/tmp"}))
	runner.WithVariable("Root", "/")

	task1 := taskpkg.NewTask()
	task1.Context = "local"
	task1.ExportAs = "EXPORT_NAME"

	task1.Commands = []string{"echo 'taskctl'"}
	task1.Name = "some test task"
	task1.Dir = "{{.Root}}"
	task1.After = []string{"echo 'after task1'"}

	d := 1 * time.Minute
	task2 := taskpkg.NewTask()
	task2.Timeout = &d
	task2.Variations = []map[string]string{{"GOOS": "windows"}, {"GOOS": "linux"}}

	task2.Commands = []string{"false"}
	task2.Name = "some test task"
	task2.Dir = "{{.Root}}"
	task2.Interactive = true

	task3 := taskpkg.NewTask()
	task3.Condition = "exit 1"

	cases := []struct {
		t                *taskpkg.Task
		skipped, errored bool
		status           int16
		output           string
	}{
		{t: task1, output: "taskctl"},
		{t: task2, status: 1, errored: true},
		{t: task3, status: -1, skipped: true},
	}

	for _, testCase := range cases {
		err = runner.Run(testCase.t)
		if err != nil && !testCase.errored && !testCase.skipped {
			t.Fatal(err)
		}

		if !testCase.skipped && testCase.t.Start.IsZero() {
			t.Error()
		}

		if !strings.Contains(testCase.t.Output(), testCase.output) {
			t.Error()
		}

		if testCase.errored && !testCase.t.Errored {
			t.Error()
		}

		if !testCase.errored && testCase.t.Errored {
			t.Error()
		}

		if testCase.t.ExitCode != testCase.status {
			t.Error()
		}
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
