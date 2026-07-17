package executor

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/variables"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	e, err := NewDefaultExecutor(nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	job1 := NewJobFromCommand("echo 'success'")
	to := 1 * time.Minute
	job1.Timeout = &to

	output, err := e.Execute(context.Background(), job1)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(output, []byte("success")) {
		t.Error()
	}

	job1 = NewJobFromCommand("exit 1")

	_, err = e.Execute(context.Background(), job1)
	if err == nil {
		t.Error()
	}

	if _, ok := IsExitStatus(err); !ok {
		t.Error()
	}

	job2 := NewJobFromCommand("echo {{ .Fail }}")
	_, err = e.Execute(context.Background(), job2)
	if err == nil {
		t.Error()
	}

	job3 := NewJobFromCommand("printf '%s\\nLine-2\\n' '=========== Line 1 ==================' ")
	_, err = e.Execute(context.Background(), job3)
	if err != nil {
		t.Error()
	}
}

// TestDefaultExecutor_Execute_PerJobEnv guards against the interpreter caching
// its environment across jobs. A single executor runs the commands of a task's
// linked list (one per command/variation), so each Execute must honor its own
// job.Env rather than reusing the first job's environment.
func TestDefaultExecutor_Execute_PerJobEnv(t *testing.T) {
	e, err := NewDefaultExecutor(nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"linux", "darwin", "windows"} {
		job := NewJobFromCommand("echo ${GOOS}")
		job.Env = variables.FromMap(map[string]string{"GOOS": want})

		out, err := e.Execute(context.Background(), job)
		if err != nil {
			t.Fatal(err)
		}

		if got := strings.TrimSpace(string(out)); got != want {
			t.Errorf("GOOS in command output = %q, want %q", got, want)
		}
	}
}

// TestDefaultExecutor_Execute_SharedShellState verifies that consecutive
// commands sharing the same environment keep shell state: a function defined by
// one command must be callable by the next (the executor reuses its
// interpreter while the environment is unchanged).
func TestDefaultExecutor_Execute_SharedShellState(t *testing.T) {
	e, err := NewDefaultExecutor(nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.Execute(context.Background(), NewJobFromCommand(`function greet() { echo "BBB"; }`)); err != nil {
		t.Fatal(err)
	}

	out, err := e.Execute(context.Background(), NewJobFromCommand("greet"))
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(string(out)); got != "BBB" {
		t.Errorf("output = %q, want %q", got, "BBB")
	}
}

// TestDefaultExecutor_Execute_PerJobDir verifies that a job's working directory
// is honored even when a same-env job ran before it: the interpreter must be
// rebuilt when job.Dir changes rather than reusing the previous directory.
func TestDefaultExecutor_Execute_PerJobDir(t *testing.T) {
	e, err := NewDefaultExecutor(nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	dirA := t.TempDir()
	dirB := t.TempDir()

	for _, want := range []string{dirA, dirB} {
		job := NewJobFromCommand("pwd")
		job.Dir = want

		out, err := e.Execute(context.Background(), job)
		if err != nil {
			t.Fatal(err)
		}

		if got := strings.TrimSpace(string(out)); got != want {
			t.Errorf("pwd = %q, want %q", got, want)
		}
	}
}
