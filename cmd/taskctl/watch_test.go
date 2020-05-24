package main

import (
	"testing"
	"time"
)

func Test_watchCommand(t *testing.T) {
	app := makeTestApp(t)
	listenSignals()

	tests := []appTest{
		{args: []string{"", "-c", "testdata/watch.yaml", "watch", "watch:watcher99"}, errored: true},
		{args: []string{"", "-c", "testdata/watch.yaml", "watch", "watch:watcher1"}, errored: false},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
		time.AfterFunc(500*time.Millisecond, func() {
			abort()
		})
	}
}
