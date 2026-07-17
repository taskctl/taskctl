package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/schema"
	"github.com/taskctl/taskctl/internal/output"
)

var showTmpl = `
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

func newShowCommand() *cli.Command {
	cmd := &cli.Command{
		Name:      "show",
		Usage:     "shows task's details",
		ArgsUsage: "show (TASK)",
		Action: func(c *cli.Context) (err error) {
			name := c.Args().First()

			if cfg.Output == output.FormatJSON {
				return encodeShowJSON(name)
			}

			t := cfg.Tasks[name]
			if t == nil {
				return errors.New("unknown task")
			}

			tmpl := template.Must(template.New("show").Parse(showTmpl))
			err = tmpl.Execute(os.Stdout, t)
			return err
		},
	}

	return cmd
}

// encodeShowJSON writes the schema_version-tagged detail document for
// `taskctl --output json show <name>`. Tasks are checked first, then
// pipelines, matching the plan's lookup order.
func encodeShowJSON(name string) error {
	if t, ok := cfg.Tasks[name]; ok {
		return json.NewEncoder(os.Stdout).Encode(struct {
			SchemaVersion int               `json:"schema_version"`
			Task          schema.TaskDetail `json:"task"`
		}{1, schema.NewTaskDetail(t)})
	}

	if g, ok := cfg.Pipelines[name]; ok {
		return json.NewEncoder(os.Stdout).Encode(struct {
			SchemaVersion int                   `json:"schema_version"`
			Pipeline      schema.PipelineDetail `json:"pipeline"`
		}{1, schema.NewPipelineDetail(name, g)})
	}

	return fmt.Errorf("unknown task or pipeline %s", name)
}
