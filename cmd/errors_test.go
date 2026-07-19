package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// present is tested in-package (not cmd_test) because it, and the error types
// it classifies, are unexported.
func TestPresent(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		want     []string
		absent   []string
	}{
		{"usage/missing-arg", []string{"show"}, 2, []string{"Error:", "show requires exactly one", "Usage:"}, nil},
		{"usage/unknown-flag", []string{"--bogus"}, 2, []string{"Error:", "unknown flag", "Usage:"}, nil},
		{"runtime/missing-config", []string{"-c", "testdata/does-not-exist.yaml", "show", "graph:task1"}, 1, []string{"Error:"}, []string{"Usage:"}},
		{"runtime/unknown-target", []string{"-c", "testdata/graph.yaml", "show", "nope"}, 1, []string{"Error:", `unknown task or pipeline "nope"`}, []string{"Usage:"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			root := NewRootCommand("test")
			root.SetOut(&buf)
			root.SetErr(&buf)
			root.SetArgs(tt.args)

			c, err := root.ExecuteC()
			if code := present(c, err); code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}

			out := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("%q not found in %q", w, out)
				}
			}
			for _, a := range tt.absent {
				if strings.Contains(out, a) {
					t.Errorf("%q unexpectedly found in %q", a, out)
				}
			}
		})
	}
}

// A failed run is already reported by the end-of-run summary, so present must
// add nothing and just carry the non-zero exit code.
func TestPresentReportedError(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	var buf bytes.Buffer
	root := NewRootCommand("test")
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--output", "prefixed", "-c", "testdata/failing.yaml", "boom"})

	c, runErr := root.ExecuteC()

	os.Stdout = origStdout
	_ = w.Close()
	_ = r.Close()

	if _, ok := errors.AsType[reportedError](runErr); !ok {
		t.Fatalf("expected reportedError, got %v", runErr)
	}
	if code := present(c, runErr); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if buf.Len() != 0 {
		t.Errorf("expected present to write nothing, got %q", buf.String())
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"plain", errors.New("boom"), 1},
		{"exit", exitError{2}, 2},
		{"wrapped-exit", fmt.Errorf("outer: %w", exitError{2}), 2},
	}
	for _, tt := range tests {
		if got := ExitCode(tt.err); got != tt.want {
			t.Errorf("%s: ExitCode = %d, want %d", tt.name, got, tt.want)
		}
	}
}
