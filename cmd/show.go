package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/schema"
)

var showTaskTmpl = `
  Name: {{ .Name -}}
{{ if .Description }}
  Description: {{ .Description }}
{{- end }}
  Context: {{ .Context }}
  Commands:
{{- range .Commands }}
    - {{ . -}}
{{ end -}}
{{ if .Dir }}
  Dir: {{ .Dir }}
{{- end }}
{{ if .Timeout }}
  Timeout: {{ .Timeout }}
{{- end}}
  AllowFailure: {{ .AllowFailure }}
`

var showPipelineTmpl = `
  Pipeline: {{ .Name }}
  Stages:
{{- range .Stages }}
    - {{ .Name }}{{ if .DependsOn }} (depends on: {{ range $i, $d := .DependsOn }}{{ if $i }}, {{ end }}{{ $d }}{{ end }}){{ end }}
{{- end }}
`

func newShowCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:               "show TASK_OR_PIPELINE",
		Short:             "shows a task's or pipeline's details",
		GroupID:           groupInspect,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: targetCompletion(cfg),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			if t := cfg.Tasks[name]; t != nil {
				if cfg.Output == output.FormatJSON {
					// Mirror the compiler's precedence: task variables override config ones.
					vars := cfg.Variables.Merge(t.Variables).Map()
					return json.NewEncoder(os.Stdout).Encode(struct {
						SchemaVersion int               `json:"schema_version"`
						Task          schema.TaskDetail `json:"task"`
					}{1, schema.NewTaskDetail(t, vars)})
				}
				return template.Must(template.New("show").Parse(showTaskTmpl)).Execute(os.Stdout, t)
			}

			if g := cfg.Pipelines[name]; g != nil {
				detail := schema.NewPipelineDetail(name, g)
				if cfg.Output == output.FormatJSON {
					return json.NewEncoder(os.Stdout).Encode(struct {
						SchemaVersion int                   `json:"schema_version"`
						Pipeline      schema.PipelineDetail `json:"pipeline"`
					}{1, detail})
				}
				return template.Must(template.New("show").Parse(showPipelineTmpl)).Execute(os.Stdout, detail)
			}

			return fmt.Errorf("unknown task or pipeline %q", name)
		},
	}
}
