package schema

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/scheduler"
	"github.com/taskctl/taskctl/task"
)

func buildTestTask() *task.Task {
	t := task.NewTask()
	t.Name = "build"
	t.Description = "builds the project"
	t.Context = "local"
	t.Commands = []string{"go build ./..."}
	t.Dir = "."
	t.AllowFailure = true
	t.Condition = "true"
	t.Env = t.Env.With("FOO", "bar")
	t.Variables = t.Variables.With("VAR1", "value1")
	timeout := 5 * time.Second
	t.Timeout = &timeout

	return t
}

func buildTestGraph(t *testing.T) *scheduler.ExecutionGraph {
	buildTask := buildTestTask()

	g, err := scheduler.NewExecutionGraph(
		&scheduler.Stage{Name: "format"},
		&scheduler.Stage{Name: "build", Task: buildTask, DependsOn: []string{"format"}},
	)
	if err != nil {
		t.Fatalf("failed to build execution graph: %v", err)
	}

	return g
}

func TestNewTaskSummary(t *testing.T) {
	tk := buildTestTask()
	summary := NewTaskSummary(tk)

	if summary.Name != "build" || summary.Description != "builds the project" || summary.Context != "local" {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"name", "description", "context"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected snake_case key %q in %s", key, data)
		}
	}
}

func TestNewTaskDetail(t *testing.T) {
	tk := buildTestTask()
	detail := NewTaskDetail(tk)

	if detail.Name != "build" {
		t.Errorf("expected name %q, got %q", "build", detail.Name)
	}
	if len(detail.Commands) != 1 || detail.Commands[0] != "go build ./..." {
		t.Errorf("unexpected commands: %+v", detail.Commands)
	}
	if detail.Env["FOO"] != "bar" {
		t.Errorf("expected env FOO=bar, got %+v", detail.Env)
	}
	if detail.Variables["VAR1"] != "value1" {
		t.Errorf("expected variable VAR1=value1, got %+v", detail.Variables)
	}
	if detail.TimeoutSeconds == nil || *detail.TimeoutSeconds != 5 {
		t.Errorf("expected timeout_seconds 5, got %+v", detail.TimeoutSeconds)
	}
	if !detail.AllowFailure {
		t.Errorf("expected allow_failure true")
	}
	if detail.Condition != "true" {
		t.Errorf("expected condition \"true\", got %q", detail.Condition)
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"name", "description", "context", "commands", "env", "variables", "dir", "timeout_seconds", "allow_failure", "condition"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected snake_case key %q in %s", key, data)
		}
	}
}

func TestNewTaskDetailOmitsEmptyOptionalFields(t *testing.T) {
	tk := task.NewTask()
	tk.Name = "noop"

	detail := NewTaskDetail(tk)

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"timeout_seconds", "description", "context", "dir", "condition"} {
		if _, ok := m[key]; ok {
			t.Errorf("expected key %q to be omitted, got %s", key, data)
		}
	}
}

func TestNewPipelineDetail(t *testing.T) {
	g := buildTestGraph(t)
	detail := NewPipelineDetail("mypipeline", g)

	if detail.Name != "mypipeline" {
		t.Errorf("expected name %q, got %q", "mypipeline", detail.Name)
	}
	if len(detail.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(detail.Stages))
	}
	if detail.Stages[0].Name != "build" || detail.Stages[1].Name != "format" {
		t.Fatalf("expected stages sorted by name, got %+v", detail.Stages)
	}
	if detail.Stages[0].Task != "build" {
		t.Errorf("expected build stage task to be %q, got %q", "build", detail.Stages[0].Task)
	}
	if len(detail.Stages[0].DependsOn) != 1 || detail.Stages[0].DependsOn[0] != "format" {
		t.Errorf("expected build stage to depend on format, got %+v", detail.Stages[0].DependsOn)
	}
	if detail.Stages[1].Task != "" {
		t.Errorf("expected format stage task to be empty, got %q", detail.Stages[1].Task)
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, ok := m["stages"]; !ok {
		t.Errorf("expected snake_case key \"stages\" in %s", data)
	}
}

func TestNewPipelineSummary(t *testing.T) {
	g := buildTestGraph(t)
	summary := NewPipelineSummary("mypipeline", g)

	if summary.Name != "mypipeline" {
		t.Errorf("expected name %q, got %q", "mypipeline", summary.Name)
	}
	if len(summary.Stages) != 2 || summary.Stages[0] != "build" || summary.Stages[1] != "format" {
		t.Fatalf("expected sorted stage names [build format], got %+v", summary.Stages)
	}
}

func TestListResponseMarshalsEmptySlicesAsArrays(t *testing.T) {
	resp := ListResponse{
		SchemaVersion: 1,
		Tasks:         []TaskSummary{},
		Pipelines:     []PipelineSummary{},
		Contexts:      []string{},
		Watchers:      []string{},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	expectedFragments := []string{
		`"schema_version":1`,
		`"tasks":[]`,
		`"pipelines":[]`,
		`"contexts":[]`,
		`"watchers":[]`,
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(string(data), fragment) {
			t.Errorf("expected %s to contain %q", data, fragment)
		}
	}
}
