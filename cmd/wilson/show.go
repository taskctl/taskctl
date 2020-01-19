package main

import (
	"errors"
	"github.com/spf13/cobra"
	"os"
	"text/template"
)

var showTmpl = `
  Name: {{ .Name -}}
{{ if .Description }}
  Description: {{ .Description }}
{{- end }}
  Context: {{ .Context }}
  Commands: 
{{- range .Command }}
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

func NewShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = loadConfig()
			if err != nil {
				return err
			}

			t, ok := tasks[args[0]]
			if !ok {
				return errors.New("unknown task")
			}

			tmpl := template.Must(template.New("show").Parse(showTmpl))
			err = tmpl.Execute(os.Stdout, t)
			return err
		},
	}

	return cmd
}
