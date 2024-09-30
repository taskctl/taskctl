package cmd

import (
	"fmt"
	"html/template"

	"github.com/spf13/cobra"
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

func newShowCmd(rootCmd *TaskCtlCmd) {

	showCmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{},
		Short:   `shows task's details`,
		Args:    cobra.RangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}

			t := conf.Tasks[args[0]]
			if t != nil {
				tmpl := template.Must(template.New("show").Parse(showTmpl))
				return tmpl.Execute(rootCmd.ChannelOut, t)
			}
			return fmt.Errorf("%s. %w", args[0], ErrIncorrectPipelineTaskArg)
		},
	}
	rootCmd.Cmd.AddCommand(showCmd)
}
