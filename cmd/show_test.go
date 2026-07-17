package cmd_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/taskctl/taskctl/internal/schema"
)

func Test_showCommand(t *testing.T) {
	app := makeTestApp()

	tests := []appTest{
		{args: []string{"", "-c", "testdata/graph.yaml", "show"}, errored: true},
		{args: []string{"", "-c", "testdata/graph.yaml", "show", "graph:task1"}, output: []string{"Name: graph:task1", "echo 'hello, world!'"}},
	}

	for _, v := range tests {
		runAppTest(app, v, t)
	}
}

func Test_showCommand_json_task(t *testing.T) {
	out, err := captureStdout(t, []string{"", "-c", "testdata/graph.yaml", "-o", "json", "show", "graph:task1"})
	if err != nil {
		t.Fatal(err)
	}

	var resp struct {
		SchemaVersion int               `json:"schema_version"`
		Task          schema.TaskDetail `json:"task"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, out)
	}

	if resp.SchemaVersion != 1 {
		t.Errorf("expected schema_version 1, got %d", resp.SchemaVersion)
	}

	if resp.Task.Name != "graph:task1" {
		t.Errorf("expected task name graph:task1, got %q", resp.Task.Name)
	}

	if len(resp.Task.Commands) == 0 {
		t.Errorf("expected commands to be populated")
	}
}

func Test_showCommand_json_pipeline(t *testing.T) {
	out, err := captureStdout(t, []string{"", "-c", "testdata/graph.yaml", "-o", "json", "show", "graph:pipeline1"})
	if err != nil {
		t.Fatal(err)
	}

	var resp struct {
		SchemaVersion int                   `json:"schema_version"`
		Pipeline      schema.PipelineDetail `json:"pipeline"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, out)
	}

	if resp.SchemaVersion != 1 {
		t.Errorf("expected schema_version 1, got %d", resp.SchemaVersion)
	}

	if resp.Pipeline.Name != "graph:pipeline1" {
		t.Errorf("expected pipeline name graph:pipeline1, got %q", resp.Pipeline.Name)
	}

	if len(resp.Pipeline.Stages) == 0 {
		t.Errorf("expected stages to be populated")
	}
}

func Test_showCommand_json_unknown(t *testing.T) {
	_, err := captureStdout(t, []string{"", "-c", "testdata/graph.yaml", "-o", "json", "show", "nope"})
	if err == nil {
		t.Fatal("expected error for unknown task or pipeline")
	}

	if !strings.Contains(err.Error(), `unknown task or pipeline "nope"`) {
		t.Errorf("unexpected error message: %v", err)
	}
}
