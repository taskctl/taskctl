// Package schema defines the JSON structures shared by taskctl's machine-readable
// CLI surface (list/show discovery documents and, eventually, run event streams).
package schema

import (
	"fmt"
	"maps"
	"slices"

	"github.com/taskctl/taskctl/internal/collections"
	"github.com/taskctl/taskctl/internal/tmpl"
	"github.com/taskctl/taskctl/scheduler"
	"github.com/taskctl/taskctl/task"
)

// ListResponse is the top-level document produced by `taskctl --output json list`.
type ListResponse struct {
	SchemaVersion int               `json:"schema_version"`
	Tasks         []TaskSummary     `json:"tasks"`
	Pipelines     []PipelineSummary `json:"pipelines"`
	Contexts      []string          `json:"contexts"`
	Watchers      []string          `json:"watchers"`
}

// TaskSummary is a brief description of a task, as listed by `list`.
type TaskSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// PipelineSummary is a brief description of a pipeline, as listed by `list`.
type PipelineSummary struct {
	Name   string   `json:"name"`
	Stages []string `json:"stages"`
}

// TaskDetail is the full description of a task, as produced by `taskctl --output json show`.
type TaskDetail struct {
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Context        string            `json:"context,omitempty"`
	Commands       []string          `json:"commands"`
	Env            map[string]string `json:"env"`
	Variables      map[string]string `json:"variables"`
	Dir            string            `json:"dir,omitempty"`
	TimeoutSeconds *float64          `json:"timeout_seconds,omitempty"`
	AllowFailure   bool              `json:"allow_failure"`
	Condition      string            `json:"condition,omitempty"`
}

// PipelineDetail is the full description of a pipeline, as produced by `taskctl --output json show`.
type PipelineDetail struct {
	Name   string        `json:"name"`
	Stages []StageDetail `json:"stages"`
}

// StageDetail describes a single stage within a pipeline's execution graph.
// Exactly one of Task or Pipeline is set: Task names the task the stage runs,
// Pipeline marks the stage as a nested sub-pipeline.
type StageDetail struct {
	Name         string   `json:"name"`
	Task         string   `json:"task,omitempty"`
	Pipeline     string   `json:"pipeline,omitempty"`
	DependsOn    []string `json:"depends_on"`
	Condition    string   `json:"condition,omitempty"`
	AllowFailure bool     `json:"allow_failure"`
}

// NewTaskSummary builds a TaskSummary from a task.Task.
func NewTaskSummary(t *task.Task) TaskSummary {
	return TaskSummary{
		Name:        t.Name,
		Description: t.Description,
		Context:     t.Context,
	}
}

// NewTaskDetail builds a TaskDetail from a task.Task. vars carries the
// config-level variables (e.g. Root, TempDir) merged under the task's own,
// used to render templated fields such as dir; templates that need runtime
// values are left as-is.
func NewTaskDetail(t *task.Task, vars map[string]any) TaskDetail {
	detail := TaskDetail{
		Name:         t.Name,
		Description:  t.Description,
		Context:      t.Context,
		Commands:     collections.OrEmpty(t.Commands),
		Env:          stringifyMap(t.Env.Map()),
		Variables:    stringifyMap(t.Variables.Map()),
		Dir:          renderOrRaw(t.Dir, vars),
		AllowFailure: t.AllowFailure,
		Condition:    t.Condition,
	}

	if t.Timeout != nil {
		seconds := t.Timeout.Seconds()
		detail.TimeoutSeconds = &seconds
	}

	return detail
}

// NewPipelineDetail builds a PipelineDetail from a pipeline's execution graph.
func NewPipelineDetail(name string, g *scheduler.ExecutionGraph) PipelineDetail {
	nodes := g.Nodes()
	names := sortedStageNames(nodes)

	stages := make([]StageDetail, 0, len(names))
	for _, stageName := range names {
		stage := nodes[stageName]

		taskName := ""
		if stage.Task != nil {
			taskName = stage.Task.Name
		}

		// A stage runs either a task or a nested sub-pipeline; the stage name
		// is the sub-pipeline's name in the latter case.
		pipelineName := ""
		if stage.Pipeline != nil {
			pipelineName = stage.Name
		}

		stages = append(stages, StageDetail{
			Name:         stage.Name,
			Task:         taskName,
			Pipeline:     pipelineName,
			DependsOn:    collections.OrEmpty(stage.DependsOn),
			Condition:    stage.Condition,
			AllowFailure: stage.AllowFailure,
		})
	}

	return PipelineDetail{
		Name:   name,
		Stages: stages,
	}
}

// NewPipelineSummary builds a PipelineSummary from a pipeline's execution graph.
func NewPipelineSummary(name string, g *scheduler.ExecutionGraph) PipelineSummary {
	nodes := g.Nodes()

	return PipelineSummary{
		Name:   name,
		Stages: sortedStageNames(nodes),
	}
}

func sortedStageNames(nodes map[string]*scheduler.Stage) []string {
	return slices.Sorted(maps.Keys(nodes))
}

// renderOrRaw renders s as a template with vars, falling back to the raw
// string when rendering fails (e.g. the template needs runtime-only values).
func renderOrRaw(s string, vars map[string]any) string {
	rendered, err := tmpl.RenderString(s, vars)
	if err != nil {
		return s
	}

	return rendered
}

func stringifyMap(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = fmt.Sprint(v)
	}

	return result
}
