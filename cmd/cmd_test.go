package cmd_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/cmd"
	"github.com/taskctl/taskctl/internal/iox"
)

type appTest struct {
	args        []string
	errored     bool
	output      []string
	absent      []string
	exactOutput string
	stdin       io.ReadCloser
	cancelAfter time.Duration
}

// runAppTest executes the root command against test.args and asserts on
// captured stdout. A fresh command is built per call because cobra commands
// carry parsed flag state and cannot be re-executed.
func runAppTest(t *testing.T, test appTest) {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Error(err)
		return
	}
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	if test.stdin != nil {
		origStdin := cmd.Stdin()
		cmd.SetStdin(test.stdin)
		defer cmd.SetStdin(origStdin)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if test.cancelAfter > 0 {
		time.AfterFunc(test.cancelAfter, cancel)
	}

	root := cmd.NewRootCommand("test")
	root.SetArgs(test.args)
	runErr := root.ExecuteContext(ctx)
	if runErr != nil && !test.errored {
		os.Stdout = origStdout
		t.Fatal(runErr)
		return
	}

	os.Stdout = origStdout
	iox.Close(w)

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		t.Error(err)
		return
	}

	s := buf.String()
	for _, v := range test.output {
		if !strings.Contains(s, v) {
			t.Errorf("%q not found in %q", v, s)
		}
	}

	for _, v := range test.absent {
		if strings.Contains(s, v) {
			t.Errorf("%q unexpectedly found in %q", v, s)
		}
	}

	if test.exactOutput != "" && s != test.exactOutput {
		t.Errorf("output mismatch, expected = %s, got = %s", test.exactOutput, s)
	}
}

// stdinLines returns a non-TTY reader holding the given lines. huh runs in
// accessible (line-based) mode against a non-terminal input, so a prompt
// consumes one line per field: a 1-based option number for a select, "y"/"n"
// for a confirm.
func stdinLines(t *testing.T, lines ...string) *os.File {
	t.Helper()
	tmpfile, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range lines {
		if _, err := tmpfile.WriteString(line + "\n"); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	return tmpfile
}

// TestCompletion exercises the dynamic target completion: cobra's __complete
// command invokes the root ValidArgsFunction, which loads the config and
// returns task and pipeline names.
func TestCompletion(t *testing.T) {
	runAppTest(t, appTest{
		args:   []string{"__complete", "-c", "testdata/graph.yaml", ""},
		output: []string{"graph:task1", "graph:pipeline1"},
	})
}

// Completion runs before PersistentPreRunE, so it cannot rely on env-var flag
// binding; it must read TASKCTL_CONFIG_FILE itself.
func TestCompletionViaConfigEnv(t *testing.T) {
	t.Setenv("TASKCTL_CONFIG_FILE", "testdata/graph.yaml")
	runAppTest(t, appTest{
		args:   []string{"__complete", ""},
		output: []string{"graph:task1", "graph:pipeline1"},
	})
}

func TestInvalidBoolEnvErrors(t *testing.T) {
	t.Setenv("TASKCTL_DEBUG", "maybe")
	runAppTest(t, appTest{
		args:    []string{"-c", "testdata/graph.yaml", "list"},
		errored: true,
	})
}

// A command-line --raw must win over TASKCTL_OUTPUT_FORMAT: raw output stays
// clean rather than falling back to the env var's format.
func TestRawFlagBeatsOutputEnv(t *testing.T) {
	t.Setenv("TASKCTL_OUTPUT_FORMAT", "json")
	runAppTest(t, appTest{
		args:        []string{"--raw", "-c", "testdata/graph.yaml", "graph:task1"},
		exactOutput: "hello, world!\n",
	})
}

// With no --output/--raw flag, TASKCTL_OUTPUT_FORMAT still selects the format.
func TestOutputEnvSelectsFormat(t *testing.T) {
	t.Setenv("TASKCTL_OUTPUT_FORMAT", "json")
	runAppTest(t, appTest{
		args:   []string{"-c", "testdata/graph.yaml", "list"},
		output: []string{`"schema_version"`, "graph:task1"},
	})
}

func TestCustomOutputFormat(t *testing.T) {
	tests := []appTest{
		{
			args:   []string{"-c", "testdata/output-none.yaml", "task1"},
			output: []string{"task1", "hello, world!", "Running task task1", "task1 finished"},
		},
		{
			args:        []string{"-c", "testdata/output-raw.yaml", "task1"},
			exactOutput: "hello, world!\n",
		},
		{
			args:   []string{"-c", "testdata/output-raw.yaml", "--output", "prefixed", "task1"},
			output: []string{"task1", "hello, world!", "Running task task1", "task1 finished"},
		},
	}

	for _, v := range tests {
		runAppTest(t, v)
	}
}

func TestRootAction(t *testing.T) {
	tests := []appTest{
		// No target and a non-TTY stdin: taskctl refuses to guess rather than
		// blocking on or silently running the interactive selector.
		{args: []string{}, errored: true},
		// An explicitly named config file that is missing is fatal.
		{args: []string{"-c", "testdata/does-not-exist.yaml", "graph:task2"}, errored: true},

		{args: []string{"--raw", "-c", "testdata/graph.yaml", "graph:task1"}, exactOutput: "hello, world!\n"},
		{
			args:   []string{"--output=prefixed", "-c", "testdata/graph.yaml", "graph:pipeline1"},
			output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"},
		},
	}

	for _, v := range tests {
		runAppTest(t, v)
	}
}

// --no-input forces non-interactive mode, so a bare invocation with no target
// errors instead of opening the selector.
func TestRootAction_NoInputFlagBlocksPrompt(t *testing.T) {
	runAppTest(t, appTest{
		args:    []string{"--no-input"},
		errored: true,
	})
}

// With a non-TTY stdout, the default dashboard degrades to prefixed output
// rather than failing to render, so task output still appears.
func TestDefaultDegradesWhenStdoutNotTTY(t *testing.T) {
	runAppTest(t, appTest{
		args:   []string{"--output", "default", "-c", "testdata/graph.yaml", "graph:task1"},
		output: []string{"hello, world!"},
	})
}

// TestFlagsInterspersedWithArgs covers the parsing bugs that motivated the
// cobra migration: persistent flags placed after a subcommand or after a
// target used to be rejected or swallowed as a target name.
func TestFlagsInterspersedWithArgs(t *testing.T) {
	tests := []appTest{
		// --output after the list subcommand (was: flag provided but not defined).
		{args: []string{"-c", "testdata/graph.yaml", "list", "--output", "json"}, output: []string{`"schema_version"`, "graph:task1"}},
		// --raw after a bare target (was: unknown task or pipeline "--raw").
		{args: []string{"-c", "testdata/graph.yaml", "graph:task1", "--raw"}, exactOutput: "hello, world!\n"},
		// --summary=false after a bare target is honored (was: swallowed).
		{args: []string{"--output=prefixed", "-c", "testdata/graph.yaml", "graph:task1", "--summary=false"}, output: []string{"hello, world!"}, absent: []string{"succeeded", "total"}},
	}

	for _, v := range tests {
		runAppTest(t, v)
	}
}
