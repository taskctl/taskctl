package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/util"
	"log"
	"os"
	"text/template"
)

var listTmpl = `Contexts:{{range $context := .Contexts}}
- {{ $context }}{{else}} no contexts {{end}}

Pipelines:
{{-range $pipeline := .Pipelines}}
- {{ $pipeline }}{{else}} no pipelines 
{{end}}

Tasks:
{{- range $task := .Tasks}}
- {{ $task }}{{else}} no tasks 
{{end}}

Watchers:
{{-range $watcher := .Watchers}}
- {{ $watcher }}{{else}} no watchers 
{{end}}
`

func NewListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List contexts, pipelines, tasks and watchers",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			t := template.Must(template.New("list").Parse(listTmpl))

			data := struct {
				Contexts, Pipelines, Tasks, Watchers []string
			}{
				Contexts:  util.ListNames(cfg.Contexts),
				Pipelines: util.ListNames(cfg.Pipelines),
				Tasks:     util.ListNames(cfg.Tasks),
				Watchers:  util.ListNames(cfg.Watchers),
			}

			err := t.Execute(os.Stdout, data)
			if err != nil {
				log.Println("executing template:", err)
			}
		},
	}
}
