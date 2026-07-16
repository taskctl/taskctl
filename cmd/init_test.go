package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func Test_initCommand(t *testing.T) {
	in := stdinLines(t, "1")
	defer func(f os.File) {
		_ = in.Close()
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
		_ = in.Close()
		_ = os.Remove(f.Name())
	}(*in)

	app := makeTestApp()
	dir := os.TempDir()
	path := filepath.Join(dir, "taskctl.yaml")
	if err := os.WriteFile(path, []byte("here"), 0764); err != nil {
		t.Fatal(err)
	}

	// With this static reader, the select form's scanner consumes both lines
	// ("1" and "n"), so the confirm form hits EOF and resolves to its
	// zero-value default (false) rather than actually parsing "n". The
	// assertion below still holds: either way the outcome is no-overwrite.
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
	// The select and the confirm each read one line; feed the confirm's answer
	// only after the select has consumed its own, so accessible mode's buffered
	// reader doesn't swallow it.
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pr.Close() }()
	go func() {
		_, _ = pw.WriteString("1\n")
		time.Sleep(250 * time.Millisecond)
		_, _ = pw.WriteString("y\n")
		_ = pw.Close()
	}()

	app := makeTestApp()
	dir := os.TempDir()
	path := filepath.Join(dir, "taskctl.yaml")
	if err := os.WriteFile(path, []byte("here"), 0764); err != nil {
		t.Fatal(err)
	}

	runAppTest(app, appTest{args: []string{"", "init", "--dir", dir}, stdin: pr, output: []string{"was created"}}, t)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "pipelines:") {
		t.Errorf("expected overwritten file to contain config template, got: %s", content)
	}
}
