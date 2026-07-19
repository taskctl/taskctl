package cmd_test

import (
	"testing"
	"time"
)

func Test_watchCommand_error(t *testing.T) {
	runAppTest(t, appTest{args: []string{"-c", "testdata/watch.yaml", "watch", "watch:watcher99"}, errored: true})
}

func Test_watchCommand_success(t *testing.T) {
	// The watcher blocks until the run context is cancelled; cancelAfter stands
	// in for the SIGINT that would stop it in a real session.
	runAppTest(t, appTest{
		args:        []string{"-c", "testdata/watch.yaml", "watch", "watch:watcher1"},
		cancelAfter: 100 * time.Millisecond,
	})
}
