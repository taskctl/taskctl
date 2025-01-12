package cmd_test

import (
	"github.com/taskctl/taskctl/cmd"
	"testing"
	"time"
)

func Test_watchCommand_error(t *testing.T) {
	app := makeTestApp()
	tests := []appTest{
		{args: []string{"", "-c", "testdata/watch.yaml", "watch", "watch:watcher99"}, errored: true},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
	}
}

func Test_watchCommand_success(t *testing.T) {
	app := makeTestApp()
	tests := []appTest{
		{args: []string{"", "-c", "testdata/watch.yaml", "watch", "watch:watcher1"}, errored: false},
	}

	for _, v := range tests {
		time.AfterFunc(100*time.Millisecond, cmd.Abort)
		runAppTest(app, v, t)
	}
}
