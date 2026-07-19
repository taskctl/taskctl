package runner

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/variables"

	taskpkg "github.com/taskctl/taskctl/task"
)

func TestTaskRunner(t *testing.T) {
	c := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"true"}, []string{"false"}, []string{"echo 1"}, []string{"echo 2"})

	runner, err := NewTaskRunner(
		WithContexts(map[string]*ExecutionContext{"default": defaultContext(), "local": c}),
		WithVariables(variables.FromMap(map[string]string{"Root": "/"})),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := runner.contexts["default"]; !ok {
		t.Error()
	}

	runner.Stdout, runner.Stderr = io.Discard, io.Discard

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

	task4 := taskpkg.NewTask()
	task4.Commands = []string{"function test_func() { echo \"BBB\"; } ", "test_func"}

	cases := []struct {
		t                *taskpkg.Task
		skipped, errored bool
		status           int16
		output           string
	}{
		{t: task1, output: "taskctl"},
		{t: task2, status: 1, errored: true},
		{t: task3, status: -1, skipped: true},
		{t: task4, output: "BBB"},
	}

	for _, testCase := range cases {
		err = runner.Run(testCase.t)
		if err != nil && !testCase.errored && !testCase.skipped {
			t.Fatal(err)
		}

		if !testCase.skipped && testCase.t.Start.IsZero() {
			t.Error()
		}

		if !strings.Contains(testCase.t.Stdout(), testCase.output) {
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

func TestTaskRunner_PreExecutionFailureKeepsExitCode(t *testing.T) {
	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard

	tsk := taskpkg.FromCommands("echo {{ .missing }}")
	if err = runner.Run(tsk); err == nil {
		t.Fatal("expected compilation error")
	}

	if tsk.ExitCode != -1 {
		t.Errorf("task that never executed must not report a success exit code, got %d", tsk.ExitCode)
	}

	if !tsk.Errored || tsk.Error == nil {
		t.Errorf("pre-execution failure must be recorded on the task, got Errored=%v Error=%v", tsk.Errored, tsk.Error)
	}

	runner.Finish()
}

func TestTaskRunner_DryRun(t *testing.T) {
	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.DryRun = true
	runner.Stdout, runner.Stderr = io.Discard, io.Discard

	// A command that would fail if executed proves nothing ran.
	fail := taskpkg.FromCommands("exit 1", "echo should-not-run")
	if err = runner.Run(fail); err != nil {
		t.Fatalf("dry-run must not surface a command failure: %v", err)
	}
	if fail.Errored {
		t.Error("dry-run task must not be marked errored")
	}
	if fail.End.IsZero() {
		t.Error("dry-run task must be marked completed (End set)")
	}
	if out := fail.Stdout(); out != "" {
		t.Errorf("dry-run must produce no command output, got %q", out)
	}

	// A condition that would normally skip the task is not evaluated either, so
	// the task is completed rather than skipped.
	conditional := taskpkg.FromCommands("echo hi")
	conditional.Condition = "exit 1"
	if err = runner.Run(conditional); err != nil {
		t.Fatal(err)
	}
	if conditional.Skipped {
		t.Error("dry-run must not evaluate the condition; the task is completed, not skipped")
	}

	runner.Finish()
}

func TestTaskRunner_ContextVariables(t *testing.T) {
	c := NewExecutionContext(nil, "", variables.NewVariables(), nil, nil, nil, nil)
	c.Variables = variables.FromMap(map[string]string{
		"greeting": "hello-from-context",
		"shared":   "context-wins",
	})

	runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"local": c}))
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard

	contextOnly := taskpkg.NewTask()
	contextOnly.Context = "local"
	contextOnly.Name = "context-only"
	contextOnly.Commands = []string{"echo '{{ .greeting }}'"}

	taskOverride := taskpkg.NewTask()
	taskOverride.Context = "local"
	taskOverride.Name = "task-override"
	taskOverride.Variables = variables.FromMap(map[string]string{"shared": "task-wins"})
	taskOverride.Commands = []string{"echo '{{ .shared }}'"}

	cases := []struct {
		t    *taskpkg.Task
		want string
	}{
		{t: contextOnly, want: "hello-from-context"},
		{t: taskOverride, want: "task-wins"},
	}

	for _, tc := range cases {
		if err := runner.Run(tc.t); err != nil {
			t.Fatalf("%s: %v", tc.t.Name, err)
		}
		if !strings.Contains(tc.t.Stdout(), tc.want) {
			t.Errorf("%s: output %q does not contain %q", tc.t.Name, tc.t.Stdout(), tc.want)
		}
	}

	runner.Finish()
}

// TestTaskRunner_ExportAsOverridesExternalEnv reproduces issue #88: when a task
// exports its output as an env var (exportAs) that already exists in the
// external environment, a later task in the same pipeline must see the exported
// value, not the inherited one. Pipeline stages share a single TaskRunner, so
// running two tasks on one runner mirrors that flow.
func TestTaskRunner_ExportAsOverridesExternalEnv(t *testing.T) {
	t.Setenv("VAR_NAME", "external")

	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard
	defer runner.Finish()

	producer := taskpkg.FromCommands(`printf "exported"`)
	producer.Name = "producer"
	producer.ExportAs = "VAR_NAME"
	if err := runner.Run(producer); err != nil {
		t.Fatal(err)
	}

	consumer := taskpkg.FromCommands(`printf "[%s]" "${VAR_NAME}"`)
	consumer.Name = "consumer"
	if err := runner.Run(consumer); err != nil {
		t.Fatal(err)
	}

	if got := consumer.Stdout(); !strings.Contains(got, "[exported]") {
		t.Errorf("exportAs must override external env: got %q, want to contain %q", got, "[exported]")
	}
}

func TestTaskRunner_NoEnvExportWithoutExportAs(t *testing.T) {
	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard
	defer runner.Finish()

	producer := taskpkg.FromCommands(`printf produced`)
	producer.Name = "producer"
	if err := runner.Run(producer); err != nil {
		t.Fatal(err)
	}

	consumer := taskpkg.FromCommands(`printf "[%s]" "${PRODUCER:-unset}"`)
	consumer.Name = "consumer"
	if err := runner.Run(consumer); err != nil {
		t.Fatal(err)
	}
	if got := consumer.Stdout(); !strings.Contains(got, "[unset]") {
		t.Errorf("without exportAs no env var must be exported: got %q", got)
	}
}

func TestTaskRunner_TasksStdoutVariable(t *testing.T) {
	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard
	defer runner.Finish()

	producer := taskpkg.FromCommands(`printf "produced"`)
	producer.Name = "producer"
	if err := runner.Run(producer); err != nil {
		t.Fatal(err)
	}

	consumer := taskpkg.FromCommands(`printf "[{{ .Tasks.Producer.Stdout }}]"`)
	consumer.Name = "consumer"
	if err := runner.Run(consumer); err != nil {
		t.Fatal(err)
	}

	if got := consumer.Stdout(); !strings.Contains(got, "[produced]") {
		t.Errorf(".Tasks.<Name>.Stdout must expose producer output: got %q", got)
	}
}

func TestTaskRunner_PredefinedTaskVars(t *testing.T) {
	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard
	defer runner.Finish()

	tsk := taskpkg.FromCommands(`printf "name={{ .Task.Name }} desc={{ .Task.Description }}"`)
	tsk.Name = "greet"
	tsk.Description = "greeter"
	if err := runner.Run(tsk); err != nil {
		t.Fatal(err)
	}

	if got := tsk.Stdout(); !strings.Contains(got, "name=greet") || !strings.Contains(got, "desc=greeter") {
		t.Errorf(".Task fields must render: got %q", got)
	}
}

func TestTaskRunner_ConditionSeesPredefinedVars(t *testing.T) {
	runner, err := NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}
	runner.Stdout, runner.Stderr = io.Discard, io.Discard
	defer runner.Finish()

	// The condition renders against the same merged view as commands; before the
	// fix it compiled against base vars and errored on .Task.Name with missing-key.
	tsk := taskpkg.FromCommands(`printf "ran"`)
	tsk.Name = "greet"
	tsk.Condition = `test "{{ .Task.Name }}" = "greet"`
	if err := runner.Run(tsk); err != nil {
		t.Fatal(err)
	}
	if tsk.Skipped {
		t.Error("condition referencing .Task.Name should have been met")
	}
	if got := tsk.Stdout(); !strings.Contains(got, "ran") {
		t.Errorf("task should have run, got %q", got)
	}
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
	fmt.Println(t.Stdout())
}
