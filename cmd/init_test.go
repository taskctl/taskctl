package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_initCommand(t *testing.T) {
	confirm := stdinConfirm(t, 1)
	defer func(f os.File) {
		_ = confirm.Close()
		_ = os.Remove(f.Name())
	}(*confirm)

	app := makeTestApp()
	dir := os.TempDir()
	_ = os.Remove(filepath.Join(dir, "taskctl.yaml"))

	runAppTest(app, appTest{args: []string{"", "init", "--dir", dir}, stdin: confirm}, t)
}

func TestInitCommand_Overwrite(t *testing.T) {
	confirm := stdinConfirm(t, 2)
	defer func(f os.File) {
		_ = confirm.Close()
		_ = os.Remove(f.Name())
	}(*confirm)

	app := makeTestApp()
	dir := os.TempDir()
	err := os.WriteFile(filepath.Join(dir, "taskctl.yaml"), []byte("here"), 0764)
	if err != nil {
		t.Fatal(err)
	}

	runAppTest(app, appTest{args: []string{"", "init", "--dir", dir}, stdin: confirm, errored: true}, t)
}
