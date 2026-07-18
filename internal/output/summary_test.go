package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/task"
)

func failedTask(name, stderr string, exit int16) *task.Task {
	t := &task.Task{Name: name, Errored: true, Error: errors.New("exit status 2"), ExitCode: exit}
	t.Start = time.Unix(0, 0)
	t.End = time.Unix(2, 0)
	t.Log.Stderr.WriteString(stderr)
	return t
}

func TestSummarizeTask(t *testing.T) {
	done := &task.Task{Name: "build"}
	done.Start = time.Unix(0, 0)
	done.End = time.Unix(1, 0)
	done.Log.Stdout.WriteString("compiled\n")

	skipped := &task.Task{Name: "deploy", Skipped: true}

	failed := failedTask("test", "line1\nboom: assertion failed\n", 2)

	tests := []struct {
		name       string
		task       *task.Task
		wantStatus string
		wantExit   int16
		wantBytes  int
		wantErrMsg string
		wantTail   []string
	}{
		{"done", done, "done", 0, len("compiled\n"), "", nil},
		{"skipped", skipped, "skipped", 0, 0, "", nil},
		{"failed", failed, "failed", 2, len("line1\nboom: assertion failed\n"), "exit status 2", []string{"line1", "boom: assertion failed"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeTask(tt.task)
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.ExitCode != tt.wantExit {
				t.Errorf("ExitCode = %d, want %d", got.ExitCode, tt.wantExit)
			}
			if got.OutputBytes != tt.wantBytes {
				t.Errorf("OutputBytes = %d, want %d", got.OutputBytes, tt.wantBytes)
			}
			if got.ErrMessage != tt.wantErrMsg {
				t.Errorf("ErrMessage = %q, want %q", got.ErrMessage, tt.wantErrMsg)
			}
			if strings.Join(got.LogTail, "|") != strings.Join(tt.wantTail, "|") {
				t.Errorf("LogTail = %v, want %v", got.LogTail, tt.wantTail)
			}
		})
	}
}

// A task whose flags say "done" but that never started actually failed before
// execution (context/hook/compile error paths return an error without setting
// Errored or Start) — the summary must not report it as succeeded.
func TestSummarizeTaskNeverStartedIsFailed(t *testing.T) {
	got := SummarizeTask(&task.Task{Name: "x"})
	if got.Status != "failed" {
		t.Errorf("Status = %q, want %q", got.Status, "failed")
	}
}

func TestSummarizeTaskFailedFallsBackToStdout(t *testing.T) {
	t2 := &task.Task{Name: "x", Errored: true, Error: errors.New("boom")}
	t2.Log.Stdout.WriteString("only-stdout\n")

	got := SummarizeTask(t2)
	if want := []string{"only-stdout"}; strings.Join(got.LogTail, "|") != strings.Join(want, "|") {
		t.Errorf("LogTail = %v, want %v", got.LogTail, want)
	}
}

func TestLastLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  []string
	}{
		{"empty", "", 3, nil},
		{"only newlines", "\n\n", 3, nil},
		{"fewer than n", "a\nb", 5, []string{"a", "b"}},
		{"trailing newline", "a\nb\n", 5, []string{"a", "b"}},
		{"more than n", "a\nb\nc\nd", 2, []string{"c", "d"}},
		{"single no newline", "x", 10, []string{"x"}},
		{"crlf endings", "a\r\nb\r\n", 5, []string{"a", "b"}},
		{"progress overwrites", "10%\r50%\r100%\nok\n", 5, []string{"100%", "ok"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBufferString(tt.input)
			got := lastLines(buf, tt.n)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("lastLines(%q, %d) = %v, want %v", tt.input, tt.n, got, tt.want)
			}
		})
	}
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tt := range tests {
		if got := humanizeBytes(tt.in); got != tt.want {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPrintRunSummary(t *testing.T) {
	items := []StageSummary{
		{Name: "build", Status: "done", Start: time.Unix(1, 0), Duration: time.Second},
		{Name: "test", Status: "failed", Start: time.Unix(2, 0), Duration: 3 * time.Second, ExitCode: 2, OutputBytes: 2048, ErrMessage: "exit status 2", LogTail: []string{"assertion failed"}},
		{Name: "deploy", Status: "skipped", Start: time.Unix(3, 0)},
	}

	var buf bytes.Buffer
	PrintRunSummary(&buf, items, 4*time.Second)
	out := buf.String()

	for _, want := range []string{
		"1 succeeded", "1 failed", "1 skipped", "4s total",
		"build", "test", "deploy",
		"exit 2", "2.0 KB output", "assertion failed", "skipped",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n---\n%s", want, out)
		}
	}
}

func TestPrintRunSummarySortsByStart(t *testing.T) {
	items := []StageSummary{
		{Name: "never-ran", Status: "skipped"}, // zero Start sorts last, not first
		{Name: "second", Status: "done", Start: time.Unix(2, 0), Duration: time.Second},
		{Name: "first", Status: "done", Start: time.Unix(1, 0), Duration: time.Second},
	}

	var buf bytes.Buffer
	PrintRunSummary(&buf, items, 2*time.Second)
	out := buf.String()

	if strings.Index(out, "first") > strings.Index(out, "second") {
		t.Errorf("expected 'first' before 'second' by start time\n%s", out)
	}
	if strings.Index(out, "never-ran") < strings.Index(out, "second") {
		t.Errorf("expected zero-Start 'never-ran' after started stages\n%s", out)
	}
}

// A failed row without a real exit code (nested sub-pipeline stages,
// pre-execution failures) must not claim "exit 0".
func TestPrintRunSummaryNoFabricatedExitCode(t *testing.T) {
	items := []StageSummary{
		{Name: "sub", Status: "failed", Start: time.Unix(1, 0), Duration: time.Second},
	}

	var buf bytes.Buffer
	PrintRunSummary(&buf, items, time.Second)

	if strings.Contains(buf.String(), "exit") {
		t.Errorf("expected no exit code for zero-exit failed row\n%s", buf.String())
	}
}
