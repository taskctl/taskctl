package main

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_initCommand(t *testing.T) {
	confirm := stdinConfirm(t)
	defer os.Remove(confirm.Name())

	app := makeTestApp(t)
	os.Remove(filepath.Join(os.TempDir(), "taskctl.yaml"))

	runAppTest(app, appTest{args: []string{"", "init", "--dir", os.TempDir()}, stdin: confirm}, t)
}
