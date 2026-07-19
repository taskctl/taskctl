package runner

import (
	"log/slog"
	"testing"

	"github.com/taskctl/taskctl/task"
	"github.com/taskctl/taskctl/variables"
)

func TestContext(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	c1 := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"true"}, []string{"false"}, []string{"true"}, []string{"false"})
	c2 := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"false"}, []string{"false"}, []string{"false"}, []string{"false"})

	runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"after_failed": c1, "before_failed": c2}))
	if err != nil {
		t.Fatal(err)
	}

	task1 := task.FromCommands("true")
	task1.Context = "after_failed"

	task2 := task.FromCommands("true")
	task2.Context = "before_failed"

	err = runner.Run(task1)
	if err != nil || task1.ExitCode != 0 {
		t.Fatal(err)
	}

	err = runner.Run(task2)
	if err == nil {
		t.Error()
	}

	if c2.startupError == nil || task2.ExitCode != -1 {
		t.Error()
	}

	runner.Finish()
}

func TestWithDocker(t *testing.T) {
	c := NewExecutionContext(nil, "", variables.NewVariables(), nil, nil, nil, nil, WithDocker(DockerSpec{Image: "alpine"}))
	if c.wrapper == nil {
		t.Fatal("expected wrapper to be set")
	}

	got := c.wrapper.wrap("echo hi", map[string]string{"FOO": "bar"}, "/app")
	want := `docker run --rm -e 'FOO=bar' -w /app alpine sh -c 'echo hi'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWithKubernetes(t *testing.T) {
	c := NewExecutionContext(nil, "", variables.NewVariables(), nil, nil, nil, nil, WithKubernetes(KubernetesSpec{Pod: "web"}))
	if c.wrapper == nil {
		t.Fatal("expected wrapper to be set")
	}

	got := c.wrapper.wrap("echo hi", map[string]string{"FOO": "bar"}, "/app")
	want := `kubectl exec web -- sh -c 'export FOO=bar; cd /app && echo hi'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWithSSH(t *testing.T) {
	c := NewExecutionContext(nil, "", variables.NewVariables(), nil, nil, nil, nil, WithSSH(SSHSpec{Host: "example.com", User: "deploy"}))
	if c.wrapper == nil {
		t.Fatal("expected wrapper to be set")
	}

	got := c.wrapper.wrap("echo hi", map[string]string{"FOO": "bar"}, "/app")
	want := `ssh deploy@example.com 'export FOO=bar; cd /app && echo hi'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
