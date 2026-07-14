package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/taskctl/taskctl/cmd"
	"github.com/taskctl/taskctl/internal/config"
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

func TestInitCommand_NoInputCreatesDefaultFile(t *testing.T) {
	cmd.SetStdinIsTTY(func() bool { return true })
	defer cmd.SetStdinIsTTY(nil)

	dir := t.TempDir()

	app := makeTestApp()
	runAppTest(app, appTest{args: []string{"", "--no-input", "init", "--dir", dir}}, t)

	if _, err := os.Stat(filepath.Join(dir, config.DefaultFileNames[0])); err != nil {
		t.Errorf("expected %s to be created, got error: %v", config.DefaultFileNames[0], err)
	}
}

func TestInitCommand_NoInputErrorsOnExistingFile(t *testing.T) {
	cmd.SetStdinIsTTY(func() bool { return true })
	defer cmd.SetStdinIsTTY(nil)

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, config.DefaultFileNames[0]), []byte("here"), 0764)
	if err != nil {
		t.Fatal(err)
	}

	app := makeTestApp()
	runAppTest(app, appTest{args: []string{"", "--no-input", "init", "--dir", dir}, errored: true}, t)

	content, readErr := os.ReadFile(filepath.Join(dir, config.DefaultFileNames[0]))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != "here" {
		t.Errorf("expected existing file to be left untouched, got %q", content)
	}
}
