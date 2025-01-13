package cmd_test

import (
	"bytes"
	"encoding/binary"
	"github.com/manifoldco/promptui"
	"github.com/taskctl/taskctl/cmd"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"strings"
	"testing"
)

type appTest struct {
	args        []string
	errored     bool
	output      []string
	exactOutput string
	stdin       io.ReadCloser
}

func makeTestApp() *cli.App {
	return cmd.NewApp("test")
}

func runAppTest(app *cli.App, test appTest, t *testing.T) {
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
	_ = w.Close()

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

	if test.exactOutput != "" && s != test.exactOutput {
		t.Errorf("output mismatch, expected = %s, got = %s", test.exactOutput, s)
	}
}

func stdinConfirm(t *testing.T, times int) *os.File {
	tmpfile, err := os.CreateTemp("", "confirm")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < times; i++ {
		err = binary.Write(tmpfile, binary.LittleEndian, promptui.KeyEnter)
		if err != nil {
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
	runAppTest(app, appTest{
		args:   []string{"", "-c", "testdata/graph.yaml", "--generate-bash-completion"},
		output: []string{"graph\\:task1", "graph\\:pipeline1"},
	}, t)
}

func TestRootAction(t *testing.T) {
	tests := []appTest{
		{args: []string{""}, output: []string{"Please use `Ctrl-C` to exit this program"}, errored: true},
		{args: []string{"", "-c", "--quiet", "testdata/graph.yaml", "graph:task2"}, errored: true},

		{args: []string{"", "--raw", "-c", "testdata/graph.yaml", "graph:task1"}, exactOutput: "hello, world!\n"},
		{
			args:   []string{"", "--output=prefixed", "-c", "testdata/graph.yaml", "graph:pipeline1"},
			output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"},
		},
	}

	for _, v := range tests {
		app := makeTestApp()
		runAppTest(app, v, t)
	}
}
