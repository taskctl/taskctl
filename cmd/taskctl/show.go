package main

import (
	"errors"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"
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

func NewShowCommand() *cli.Command {
	cmd := &cli.Command{
		Name:      "show",
		Usage:     "shows task's details",
		ArgsUsage: "show (TASK)",
		Action: func(c *cli.Context) (err error) {
			t, ok := tasks[c.Args().First()]
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
