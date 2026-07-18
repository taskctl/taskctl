package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/iox"
)

func Test_initCommand(t *testing.T) {
	in := stdinLines(t, "1")
	defer func(f os.File) {
		iox.Close(in)
		_ = os.Remove(f.Name())
	}(*in)

	app := makeTestApp()
	dir := os.TempDir()
	_ = os.Remove(filepath.Join(dir, "taskctl.yaml"))

	runAppTest(app, appTest{args: []string{"", "init", "--dir", dir}, stdin: in, output: []string{"was created"}}, t)
}

// TestInitCommand_NoOverwrite verifies the safe default: when the file exists
// and the confirmation isn't answered affirmatively, the file is left intact.
func TestInitCommand_NoOverwrite(t *testing.T) {
	in := stdinLines(t, "1", "n")
	defer func(f os.File) {
		iox.Close(in)
		_ = os.Remove(f.Name())
	}(*in)

	app := makeTestApp()
	dir := os.TempDir()
	path := filepath.Join(dir, "taskctl.yaml")
	if err := os.WriteFile(path, []byte("here"), 0764); err != nil {
		t.Fatal(err)
	}

	// The select consumes "1" and the confirm reads "n" (PromptReader hands
	// each prompt exactly one line), so the confirm parses "n" -> false and the
	// file is left intact.
	runAppTest(app, appTest{args: []string{"", "init", "--dir", dir}, stdin: in}, t)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "here" {
		t.Errorf("expected file to be left intact, got: %s", content)
	}
}

func TestInitCommand_Overwrite(t *testing.T) {
	// PromptReader gives each prompt exactly one line, so a plain two-line stdin
	// works: the select reads "1" and the confirm reads "y" -> overwrite.
	in := stdinLines(t, "1", "y")
	defer func(f os.File) {
		iox.Close(in)
		_ = os.Remove(f.Name())
	}(*in)

	app := makeTestApp()
	dir := os.TempDir()
	path := filepath.Join(dir, "taskctl.yaml")
	// Pre-seed a file longer than the template with a trailing marker: without
	// O_TRUNC the marker survives past the freshly-written (shorter) template.
	existing := strings.Repeat("# old\n", 500) + "TRAILING_MARKER\n"
	if err := os.WriteFile(path, []byte(existing), 0764); err != nil {
		t.Fatal(err)
	}

	runAppTest(app, appTest{args: []string{"", "init", "--dir", dir}, stdin: in, output: []string{"was created"}}, t)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "pipelines:") {
		t.Errorf("expected overwritten file to contain config template, got: %s", content)
	}
	if strings.Contains(string(content), "TRAILING_MARKER") {
		t.Errorf("expected overwrite to truncate the old file, but trailing bytes remain: %s", content)
	}
}

func TestInitCommand_NoInputCreatesDefaultFile(t *testing.T) {
	dir := t.TempDir()

	app := makeTestApp()
	runAppTest(app, appTest{args: []string{"", "--no-input", "init", "--dir", dir}}, t)

	if _, err := os.Stat(filepath.Join(dir, config.DefaultFileNames[0])); err != nil {
		t.Errorf("expected %s to be created, got error: %v", config.DefaultFileNames[0], err)
	}
}

func TestInitCommand_NoInputErrorsOnExistingFile(t *testing.T) {
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
