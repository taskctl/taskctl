package cmd

import (
	"fmt"
	"slices"
	"text/template"

	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/spf13/cobra"
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

var (
	listCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{},
		Short:   `lists contexts, pipelines, tasks and watchers`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			t := template.Must(template.New("list").Parse(listTmpl))

			contexts := utils.MapKeys(conf.Contexts)
			pipelines := utils.MapKeys(conf.Pipelines)
			tasks := utils.MapKeys(conf.Tasks)
			watchers := utils.MapKeys(conf.Watchers)

			slices.Sort(contexts)
			slices.Sort(pipelines)
			slices.Sort(tasks)
			slices.Sort(watchers)

			return t.Execute(ChannelOut, struct {
				Contexts, Pipelines, Tasks, Watchers []string
			}{
				Contexts:  contexts,
				Pipelines: pipelines,
				Tasks:     tasks,
				Watchers:  watchers,
			})
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
	listPipelines = &cobra.Command{
		Use:   "pipelines",
		Short: `lists pipelines`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range utils.MapKeys(conf.Pipelines) {
				fmt.Fprintln(ChannelOut, name)
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
	listTasks = &cobra.Command{
		Use:   "tasks",
		Short: `lists tasks`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range utils.MapKeys(conf.Tasks) {
				fmt.Fprintln(ChannelOut, name)
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
	listWatchers = &cobra.Command{
		Use:   "watchers",
		Short: `lists watchers`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range utils.MapKeys(conf.Watchers) {
				fmt.Fprintln(ChannelOut, name)
			}
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
)

func init() {
	listCmd.AddCommand(listPipelines)
	listCmd.AddCommand(listTasks)
	listCmd.AddCommand(listWatchers)
	TaskCtlCmd.AddCommand(listCmd)
}
