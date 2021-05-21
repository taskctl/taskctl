package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
)

type appTest struct {
	args        []string
	errored     bool
	output      []string
	exactOutput string
	stdin       io.ReadCloser
}

func makeTestApp(t *testing.T) *cli.App {
	return makeApp()
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
		origStdin := stdin
		stdin = test.stdin
		defer func() {
			stdin = origStdin
		}()
	}

	err = app.Run(test.args)
	os.Stdout = origStdout
	w.Close()

	if err != nil && !test.errored {
		t.Fatal(err)
		return
	}

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
	tmpfile, err := ioutil.TempFile("", "confirm")
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
	app := makeTestApp(t)
	runAppTest(app, appTest{
		args:   []string{"", "-c", "testdata/graph.yaml", "--generate-bash-completion"},
		output: []string{"graph\\:task1", "graph\\:pipeline1"},
	}, t)
}

func TestCustomOutputFormat(t *testing.T) {
	tests := []appTest{
		{
			args:   []string{"", "-c", "testdata/output-none.yaml", "task1"},
			exactOutput: "\x1b[36mtask1\x1b[0m: hello, world!\r\n",
		},
		{
			args:   []string{"", "-c", "testdata/output-raw.yaml", "task1"},
			exactOutput: "hello, world!\n",
		},
		{
			args:   []string{"", "-c", "testdata/output-raw.yaml", "--output", "prefixed", "task1"},
			exactOutput: "\x1b[36mtask1\x1b[0m: hello, world!\r\n",
		},

	}

	for _, v := range tests {
		app := makeTestApp(t)
		runAppTest(app, v, t)
	}
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
		app := makeTestApp(t)
		runAppTest(app, v, t)
	}
}

func Test_makeApp(t *testing.T) {
	tests := []struct {
		name string
		want *cli.App
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeApp(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeApp() = %v, want %v", got, tt.want)
			}
		})
	}
}
