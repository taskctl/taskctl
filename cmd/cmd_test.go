package cmd_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"

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
}

func makeTestApp() *cli.App {
	return cmd.NewApp("test")
}

func runAppTest(t *testing.T, app *cli.App, test appTest) {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Error(err)
		return
	}
	os.Stdout = w
	defer func() {
		os.Stdout = origStdout
	}()

	if test.stdin != nil {
		origStdin := cmd.Stdin()
		cmd.SetStdin(test.stdin)
		defer func() {
			cmd.SetStdin(origStdin)
		}()
	}

	err = app.Run(test.args)
	if err != nil && !test.errored {
		t.Fatal(err)
		return
	}

	os.Stdout = origStdout
	iox.Close(w)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Error(err)
		return
	}

	s := buf.String()
	if len(test.output) > 0 {
		for _, v := range test.output {
			if !strings.Contains(s, v) {
				t.Errorf("\"%s\" not found in \"%s\"", v, s)
			}
		}
	}

	for _, v := range test.absent {
		if strings.Contains(s, v) {
			t.Errorf("\"%s\" unexpectedly found in \"%s\"", v, s)
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

func TestBashComplete(t *testing.T) {
	app := makeTestApp()
	runAppTest(t, app, appTest{
		args:   []string{"", "-c", "testdata/graph.yaml", "--generate-bash-completion"},
		output: []string{"graph\\:task1", "graph\\:pipeline1"},
	})
}

func TestCustomOutputFormat(t *testing.T) {
	tests := []appTest{
		{
			args:   []string{"", "-c", "testdata/output-none.yaml", "task1"},
			output: []string{"task1", "hello, world!", "Running task task1", "task1 finished"},
		},
		{
			args:        []string{"", "-c", "testdata/output-raw.yaml", "task1"},
			exactOutput: "hello, world!\n",
		},
		{
			args:   []string{"", "-c", "testdata/output-raw.yaml", "--output", "prefixed", "task1"},
			output: []string{"task1", "hello, world!", "Running task task1", "task1 finished"},
		},
	}

	for _, v := range tests {
		app := makeTestApp()
		runAppTest(t, app, v)
	}
}

func TestRootAction(t *testing.T) {
	tests := []appTest{
		// No target and a non-TTY stdin: taskctl refuses to guess rather than
		// blocking on or silently running the interactive selector.
		{args: []string{""}, errored: true},
		{args: []string{"", "-c", "--quiet", "testdata/graph.yaml", "graph:task2"}, errored: true},

		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "graph:task1"}, exactOutput: "hello, world!\n"},
		{
			args:   []string{"", "--output=prefixed", "-c", "testdata/graph.yaml", "graph:pipeline1"},
			output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"},
		},
	}

	for _, v := range tests {
		app := makeTestApp()
		runAppTest(t, app, v)
	}
}

// --no-input forces non-interactive mode, so a bare invocation with no target
// errors instead of opening the selector.
func TestRootAction_NoInputFlagBlocksPrompt(t *testing.T) {
	app := makeTestApp()
	runAppTest(t, app, appTest{
		args:    []string{"", "--no-input"},
		errored: true,
	})
}

// With a non-TTY stdout, the default dashboard degrades to prefixed output
// rather than failing to render, so task output still appears.
func TestDefaultDegradesWhenStdoutNotTTY(t *testing.T) {
	app := makeTestApp()
	runAppTest(t, app, appTest{
		args:   []string{"", "--output", "default", "-c", "testdata/graph.yaml", "graph:task1"},
		output: []string{"hello, world!"},
	})
}
