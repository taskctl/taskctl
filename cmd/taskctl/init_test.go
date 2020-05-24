package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func Test_initCommand(t *testing.T) {
	confirm := stdinConfirm(t, 1)
	defer func(f os.File) {
		os.Remove(f.Name())
	}(*confirm)

	app := makeTestApp(t)
	os.Remove(filepath.Join(os.TempDir(), "taskctl.yaml"))

	runAppTest(app, appTest{args: []string{"", "init", "--dir", os.TempDir()}, stdin: confirm}, t)
}

func TestInitCommand_Overwrite(t *testing.T) {
	confirm := stdinConfirm(t, 2)
	defer func(f os.File) {
		os.Remove(f.Name())
	}(*confirm)

	app := makeTestApp(t)
	err := ioutil.WriteFile(filepath.Join(os.TempDir(), "taskctl.yaml"), []byte("here"), 0764)
	if err != nil {
		t.Fatal(err)
	}

	runAppTest(app, appTest{args: []string{"", "init", "--dir", os.TempDir()}, stdin: confirm, errored: true}, t)
}
