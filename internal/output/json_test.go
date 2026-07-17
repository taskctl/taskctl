package output

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/taskctl/taskctl/task"
)

// decodeLines unmarshals each newline-terminated line of b, failing the test
// if any line is not valid JSON.
func decodeLines(t *testing.T, b []byte) []map[string]any {
	t.Helper()

	var events []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("invalid ndjson line %q: %v", line, err)
		}
		events = append(events, m)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	return events
}

func TestJSONOutputWriter_HeaderAndFooter(t *testing.T) {
	var buf bytes.Buffer
	tt := task.FromCommands("echo hi")
	tt.Name = "task1"

	d := newJSONOutputWriter(tt, &buf)

	if err := d.WriteHeader(); err != nil {
		t.Fatal(err)
	}
	if err := d.WriteFooter(); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %+v", len(events), events)
	}

	if events[0]["event"] != "task_started" || events[0]["task"] != "task1" {
		t.Errorf("unexpected task_started event: %+v", events[0])
	}

	if events[1]["event"] != "task_finished" || events[1]["task"] != "task1" {
		t.Errorf("unexpected task_finished event: %+v", events[1])
	}
	if events[1]["status"] != "done" {
		t.Errorf("expected status done, got %+v", events[1]["status"])
	}
}

func TestJSONOutputWriter_MultiLineAndPartial(t *testing.T) {
	var buf bytes.Buffer
	tt := task.FromCommands("echo hi")
	tt.Name = "task1"

	d := newJSONOutputWriter(tt, &buf)

	// Two complete lines in one write.
	if _, err := d.Write([]byte("line1\nline2\n")); err != nil {
		t.Fatal(err)
	}

	// A partial line split across two writes.
	if _, err := d.Write([]byte("partial-")); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write([]byte("line\n")); err != nil {
		t.Fatal(err)
	}

	// Trailing partial remainder with no newline, flushed by WriteFooter.
	if _, err := d.Write([]byte("trailing")); err != nil {
		t.Fatal(err)
	}
	if err := d.WriteFooter(); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())

	var outputLines []string
	for _, ev := range events {
		if ev["event"] == "task_output" {
			outputLines = append(outputLines, ev["data"].(string))
			if ev["stream"] != "stdout" {
				t.Errorf("expected stdout stream, got %+v", ev["stream"])
			}
		}
	}

	expected := []string{"line1", "line2", "partial-line", "trailing"}
	if len(outputLines) != len(expected) {
		t.Fatalf("expected lines %v, got %v", expected, outputLines)
	}
	for i, v := range expected {
		if outputLines[i] != v {
			t.Errorf("expected line %d to be %q, got %q", i, v, outputLines[i])
		}
	}

	last := events[len(events)-1]
	if last["event"] != "task_finished" {
		t.Errorf("expected last event to be task_finished, got %+v", last)
	}
}

func TestJSONOutputWriter_StreamWriter_Stderr(t *testing.T) {
	var buf bytes.Buffer
	tt := task.FromCommands("echo hi")
	tt.Name = "task1"

	d := newJSONOutputWriter(tt, &buf)

	sw := d.StreamWriter("stderr")
	if _, err := sw.Write([]byte("oops\n")); err != nil {
		t.Fatal(err)
	}

	if err := d.WriteFooter(); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())
	found := false
	for _, ev := range events {
		if ev["event"] == "task_output" {
			if ev["stream"] != "stderr" {
				t.Errorf("expected stderr stream, got %+v", ev["stream"])
			}
			if ev["data"] != "oops" {
				t.Errorf("expected data oops, got %+v", ev["data"])
			}
			found = true
		}
	}
	if !found {
		t.Error("expected a task_output event for stderr")
	}
}

func TestJSONOutputWriter_FooterFlushesFailedStatus(t *testing.T) {
	var buf bytes.Buffer
	tt := task.FromCommands("false")
	tt.Name = "task1"
	tt.Errored = true
	tt.ExitCode = 1
	tt.Error = errTest{}
	tt.Log.Stderr.WriteString("boom")

	d := newJSONOutputWriter(tt, &buf)
	if err := d.WriteFooter(); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())
	last := events[len(events)-1]
	if last["status"] != "failed" {
		t.Errorf("expected status failed, got %+v", last["status"])
	}
	if last["exit_code"].(float64) != 1 {
		t.Errorf("expected exit_code 1, got %+v", last["exit_code"])
	}
	if last["error"] != "boom" {
		t.Errorf("expected error message boom, got %+v", last["error"])
	}
}

type errTest struct{}

func (errTest) Error() string { return "boom" }

func TestEmitRunStartedAndFinished(t *testing.T) {
	var buf bytes.Buffer

	if err := EmitRunStarted(&buf, []string{"pipeline1", "task1"}); err != nil {
		t.Fatal(err)
	}

	results := []TaskResult{
		{Task: "task1", Status: "done", ExitCode: 0, DurationMs: 5},
	}
	if err := EmitRunFinished(&buf, "done", 5, results, ""); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0]["event"] != "run_started" {
		t.Errorf("expected run_started, got %+v", events[0])
	}
	if events[0]["schema_version"].(float64) != 1 {
		t.Errorf("expected schema_version 1, got %+v", events[0]["schema_version"])
	}

	if events[1]["event"] != "run_finished" {
		t.Errorf("expected run_finished, got %+v", events[1])
	}
	if events[1]["status"] != "done" {
		t.Errorf("expected status done, got %+v", events[1]["status"])
	}
}

func TestEmitRunFinished_FailureCarriesErrorAndEmptyTasks(t *testing.T) {
	var buf bytes.Buffer

	if err := EmitRunFinished(&buf, "failed", 0, nil, "unknown task or pipeline nope"); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())
	ev := events[0]

	if ev["error"] != "unknown task or pipeline nope" {
		t.Errorf("expected error message in run_finished, got %+v", ev["error"])
	}
	// nil results must marshal as [], not null, so consumers can range over it.
	if tasks, ok := ev["tasks"].([]any); !ok || len(tasks) != 0 {
		t.Errorf("expected tasks to be an empty array, got %+v", ev["tasks"])
	}
}

func TestNewTaskOutput_JSON(t *testing.T) {
	var buf bytes.Buffer
	tt := task.FromCommands("echo hi")
	tt.Name = "task1"

	o, err := NewTaskOutput(tt, FormatJSON, &buf, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := o.Start(); err != nil {
		t.Fatal(err)
	}

	if _, err := o.Stdout().Write([]byte("out\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Stderr().Write([]byte("err\n")); err != nil {
		t.Fatal(err)
	}

	if err := o.Finish(); err != nil {
		t.Fatal(err)
	}

	events := decodeLines(t, buf.Bytes())

	var sawStdout, sawStderr bool
	for _, ev := range events {
		if ev["event"] != "task_output" {
			continue
		}
		switch ev["stream"] {
		case "stdout":
			sawStdout = true
		case "stderr":
			sawStderr = true
		}
	}

	if !sawStdout || !sawStderr {
		t.Errorf("expected both stdout and stderr task_output events, got %+v", events)
	}

	if tt.Log.Stdout.String() != "out\n" {
		t.Errorf("expected task log to also capture stdout, got %q", tt.Log.Stdout.String())
	}
}
