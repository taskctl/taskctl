package main

import (
	"fmt"
	"os"
	"sort"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/pkg/util"
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
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			t := template.Must(template.New("list").Parse(listTmpl))

			contexts := util.ListNames(cfg.Contexts)
			pipelines := util.ListNames(cfg.Pipelines)
			tasks := util.ListNames(cfg.Tasks)
			watchers := util.ListNames(cfg.Watchers)

			sort.Strings(contexts)
			sort.Strings(pipelines)
			sort.Strings(tasks)
			sort.Strings(watchers)

			err = t.Execute(os.Stdout, struct {
				Contexts, Pipelines, Tasks, Watchers []string
			}{
				Contexts:  contexts,
				Pipelines: pipelines,
				Tasks:     tasks,
				Watchers:  watchers,
			})
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
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			for _, name := range util.ListNames(cfg.Tasks) {
				fmt.Println(name)
			}

			return nil
		},
	}
}

func NewListPipelinesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pipelines",
		Short: "List pipelines",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			for _, name := range util.ListNames(cfg.Pipelines) {
				fmt.Println(name)
			}

			return nil
		},
	}
}

func NewListWatchersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watchers",
		Short: "List watchers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			for _, name := range util.ListNames(cfg.Watchers) {
				fmt.Println(name)
			}

			return nil
		},
	}
}
