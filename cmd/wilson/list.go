package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/util"
	"os"
	"text/template"
)

var listTmpl = `Contexts:{{range $context := .Contexts}}
- {{ $context }}{{else}} no contexts {{end}}

Pipelines:
{{- range $pipeline := .Pipelines}}
- {{ $pipeline }}{{else}} no pipelines 
{{end}}

Tasks:
{{- range $task := .Tasks}}
- {{ $task }}{{else}} no tasks 
{{end}}

Watchers:
{{- range $watcher := .Watchers}}
- {{ $watcher }}{{else}} no watchers 
{{end}}
`

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contexts, pipelines, tasks and watchers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return err
		},
	}

	cmd.AddCommand(NewListTasksCommand())
	cmd.AddCommand(NewListPipelinesCommand())
	cmd.AddCommand(NewListWatchersCommand())

	return cmd
}

func NewListTasksCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tasks",
		Short: "List tasks",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range util.ListNames(cfg.Tasks) {
				fmt.Println(name)
			}
		},
	}
}

func NewListPipelinesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pipelines",
		Short: "List pipelines",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range util.ListNames(cfg.Pipelines) {
				fmt.Println(name)
			}
		},
	}
}

func NewListWatchersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watchers",
		Short: "List watchers",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			for _, name := range util.ListNames(cfg.Watchers) {
				fmt.Println(name)
			}
		},
	}
}
