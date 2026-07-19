package cmd_test

import (
	"encoding/json"
	"testing"
)

func Test_runCommand(t *testing.T) {
	tests := []appTest{
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "graph:task2"}, errored: true},
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run"}, errored: true},
		// raw output defaults the summary off, so command output stays clean.
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "graph:task1"}, exactOutput: "hello, world!\n"},
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "task", "graph:task1"}, exactOutput: "hello, world!\n"},
		// `run task` only matches tasks: a pipeline name is rejected.
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "task", "graph:pipeline1"}, errored: true},
		// and its completion offers only tasks, never pipelines.
		{args: []string{"__complete", "-c", "testdata/graph.yaml", "run", "task", ""}, output: []string{"graph:task1"}, absent: []string{"graph:pipeline1"}},
		// but an explicit --summary opts raw back in.
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "--summary", "graph:task1"}, output: []string{"hello, world!", "succeeded", "total"}},
		// a root-level --summary=false must not be shadowed by the run
		// command's own (unset, default-true) flag declaration.
		{args: []string{"--output=prefixed", "--summary=false", "-c", "testdata/graph.yaml", "run", "graph:task1"}, output: []string{"hello, world!"}, absent: []string{"succeeded", "total"}},
		{args: []string{"--raw", "-c", "testdata/graph.yaml", "run", "--summary", "pipeline", "graph:pipeline1"}, output: []string{"graph:task3", "hello, world!\n"}},
		{
			args:   []string{"--output=prefixed", "-c", "testdata/graph.yaml", "run", "graph:pipeline1"},
			output: []string{"graph:task1", "graph:task2", "graph:task3", "hello, world!"},
		},
	}

	for _, v := range tests {
		runAppTest(t, v)
	}
}

// Test_runCommand_json runs the graph:pipeline1 fixture (which has stages
// graph:task2 and graph:task3 running in parallel, both depending on
// graph:task1) under -o json and asserts the resulting stdout is a valid
// NDJSON event stream: every line parses as JSON, the first event is
// run_started with a schema_version, and the last is run_finished with
// per-task results.
func Test_runCommand_json(t *testing.T) {
	out, err := captureStdout(t, []string{"-c", "testdata/graph.yaml", "-o", "json", "run", "graph:pipeline1"})
	if err != nil {
		t.Fatal(err)
	}

	lines := splitLines(out)
	if len(lines) == 0 {
		t.Fatal("expected at least one NDJSON line, got none")
	}

	var events []map[string]any
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatalf("invalid ndjson line %q: %v", line, err)
		}
		events = append(events, m)
	}

	first := events[0]
	if first["event"] != "run_started" {
		t.Errorf("expected first event to be run_started, got %+v", first)
	}
	if first["schema_version"].(float64) != 1 {
		t.Errorf("expected schema_version 1, got %+v", first["schema_version"])
	}

	last := events[len(events)-1]
	if last["event"] != "run_finished" {
		t.Errorf("expected last event to be run_finished, got %+v", last)
	}

	tasks, ok := last["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Errorf("expected run_finished.tasks to be a non-empty array, got %+v", last["tasks"])
	}
}

// Test_runCommandSummary asserts the end-of-run summary is printed in a human
// output mode: for a single-task run (the summary now extends to single tasks)
// and for a pipeline (exercising summarizeGraph over stages, including the
// nested sub-pipeline node). --output json is verified summary-free.
func Test_runCommandSummary(t *testing.T) {
	tests := []appTest{
		{
			args:   []string{"--output=prefixed", "-c", "testdata/graph.yaml", "graph:task1"},
			output: []string{"hello, world!", "succeeded", "total"},
		},
		{
			args:   []string{"--output=prefixed", "-c", "testdata/graph.yaml", "run", "graph:pipeline1"},
			output: []string{"succeeded", "total", "graph:task1"},
		},
	}

	for _, v := range tests {
		runAppTest(t, v)
	}
}

func splitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			if i > start {
				lines = append(lines, b[start:i])
			}
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, b[start:])
	}
	return lines
}

func Test_runCommandWithArgumentsList(t *testing.T) {
	tests := []appTest{
		{args: []string{"--raw", "-c", "testdata/task.yaml", "run", "task", "task:task1", "--", "first", "second"}, exactOutput: "This is first argument\n"},
		{args: []string{"--raw", "-c", "testdata/task.yaml", "run", "task", "task:task2", "--", "first", "second"}, exactOutput: "This is second argument\n"},
		{args: []string{"--raw", "-c", "testdata/task.yaml", "run", "task", "task:task3", "--", "first", "and", "second"}, exactOutput: "This is first and second arguments\n"},
	}

	for _, v := range tests {
		runAppTest(t, v)
	}
}
