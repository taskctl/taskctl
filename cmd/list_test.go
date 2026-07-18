package cmd_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/taskctl/taskctl/internal/iox"
	"github.com/taskctl/taskctl/internal/schema"
)

func Test_listCommand(t *testing.T) {
	app := makeTestApp()

	tests := []appTest{
		{args: []string{"", "-c", "testdata/graph.yaml", "list"}, output: []string{"graph:pipeline1", "graph:task1", "no watchers"}},
		{args: []string{"", "-c", "testdata/graph.yaml", "list", "pipelines"}, output: []string{"graph:pipeline1"}},
		{args: []string{"", "-c", "testdata/graph.yaml", "list", "tasks"}, output: []string{"graph:task1"}},
		{args: []string{"", "-c", "testdata/graph.yaml", "list", "watchers"}, exactOutput: ""},
	}

	for _, v := range tests {
		runAppTest(t, app, v)
	}
}

// captureStdout runs the test app with the given args and returns everything
// written to stdout. Shared by list_test.go and show_test.go for JSON output
// assertions, where the string-contains helpers in runAppTest are not enough
// (JSON must be unmarshaled, never string-compared).
func captureStdout(t *testing.T, args []string) ([]byte, error) {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	app := makeTestApp()
	runErr := app.Run(args)

	os.Stdout = origStdout
	iox.Close(w)
	defer iox.Close(r)

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes(), runErr
}

func Test_listCommand_json(t *testing.T) {
	out, err := captureStdout(t, []string{"", "-c", "testdata/graph.yaml", "-o", "json", "list"})
	if err != nil {
		t.Fatal(err)
	}

	var resp schema.ListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, out)
	}

	if resp.SchemaVersion != 1 {
		t.Errorf("expected schema_version 1, got %d", resp.SchemaVersion)
	}

	if resp.Contexts == nil {
		t.Errorf("expected contexts to be [] not null")
	}

	if resp.Watchers == nil {
		t.Errorf("expected watchers to be [] not null")
	}

	foundTask := false
	for _, ts := range resp.Tasks {
		if ts.Name == "graph:task1" {
			foundTask = true
		}
	}
	if !foundTask {
		t.Errorf("expected graph:task1 in tasks, got %+v", resp.Tasks)
	}

	foundPipeline := false
	for _, p := range resp.Pipelines {
		if p.Name == "graph:pipeline1" {
			foundPipeline = true
			if len(p.Stages) == 0 {
				t.Errorf("expected graph:pipeline1 stages to be populated")
			}
		}
	}
	if !foundPipeline {
		t.Errorf("expected graph:pipeline1 in pipelines, got %+v", resp.Pipelines)
	}
}

func Test_listCommand_json_subcommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		key  string
	}{
		{"tasks", []string{"", "-c", "testdata/graph.yaml", "-o", "json", "list", "tasks"}, "tasks"},
		{"pipelines", []string{"", "-c", "testdata/graph.yaml", "-o", "json", "list", "pipelines"}, "pipelines"},
		{"watchers", []string{"", "-c", "testdata/graph.yaml", "-o", "json", "list", "watchers"}, "watchers"},
	}

	allKeys := []string{"tasks", "pipelines", "watchers", "contexts"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureStdout(t, tt.args)
			if err != nil {
				t.Fatal(err)
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(out, &raw); err != nil {
				t.Fatalf("invalid json: %v\noutput: %s", err, out)
			}

			if _, ok := raw["schema_version"]; !ok {
				t.Errorf("expected schema_version key, got %+v", raw)
			}

			if _, ok := raw[tt.key]; !ok {
				t.Errorf("expected %s key, got %+v", tt.key, raw)
			}

			for _, other := range allKeys {
				if other == tt.key {
					continue
				}
				if _, ok := raw[other]; ok {
					t.Errorf("did not expect %s key in %s response, got %+v", other, tt.key, raw)
				}
			}
		})
	}
}
