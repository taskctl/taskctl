package cmd_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	taskctlCmd "github.com/Ensono/taskctl/cmd/taskctl"
	"github.com/Ensono/taskctl/pkg/output"
)

type cmdRunTestInput struct {
	args        []string
	errored     bool
	exactOutput string
	output      []string
}

func cmdRunTestHelper(t *testing.T, testInput *cmdRunTestInput) {
	t.Helper()

	// taskctlCmd.ChannelOut = nil
	// taskctlCmd.ChannelErr = nil
	cmd := taskctlCmd.NewTaskCtlCmd()
	os.Args = append([]string{os.Args[0]}, testInput.args...)

	cmd.Cmd.SetArgs(testInput.args)
	errOut := output.NewSafeWriter(&bytes.Buffer{})
	stdOut := output.NewSafeWriter(&bytes.Buffer{})
	cmd.Cmd.SetErr(errOut)
	cmd.Cmd.SetOut(stdOut)

	if err := cmd.InitCommand(); err != nil {
		t.Fatal(err)
	}

	logOut := output.NewSafeWriter(&bytes.Buffer{})
	logErr := output.NewSafeWriter(&bytes.Buffer{})

	// silence output
	taskctlCmd.ChannelOut = logOut
	taskctlCmd.ChannelErr = logErr

	// fmt.Printf("input args: %v\n", cmdArgs)

	defer func() {
		cmd = nil
		taskctlCmd.ChannelErr = nil
		taskctlCmd.ChannelOut = nil
	}()

	if err := cmd.Execute(context.TODO()); err != nil {
		if testInput.errored {
			return
		}
		t.Fatalf("\ngot: %v\nwanted <nil>\n", err)
	}

	if testInput.errored && errOut.Len() < 1 {
		t.Errorf("\ngot: nil\nwanted an error to be thrown")
	}
	if len(testInput.output) > 0 {
		for _, v := range testInput.output {
			if !strings.Contains(logOut.String(), v) {
				t.Errorf("\ngot: %s\vnot found in: %v", logOut.String(), v)
			}
		}
	}
	if testInput.exactOutput != "" && logOut.String() != testInput.exactOutput {
		t.Errorf("output mismatch\ngot: %s\n\nwanted: %s", logOut.String(), testInput.exactOutput)
	}
}
