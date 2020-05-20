package main

import (
	"fmt"
	"os"
	"sort"
	"text/template"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/pkg/utils"
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

func newListCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "lists contexts, pipelines, tasks and watchers",
		Action: func(c *cli.Context) (err error) {
			t := template.Must(template.New("list").Parse(listTmpl))

			contexts := utils.MapKeys(cfg.Contexts)
			pipelines := utils.MapKeys(cfg.Pipelines)
			tasks := utils.MapKeys(cfg.Tasks)
			watchers := utils.MapKeys(cfg.Watchers)

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
		Subcommands: []*cli.Command{
			{
				Name:        "tasks",
				Description: "List tasks",
				Action: func(c *cli.Context) error {
					for _, name := range utils.MapKeys(cfg.Tasks) {
						fmt.Println(name)
					}

					return nil
				},
			},
			{
				Name:        "pipelines",
				Description: "List pipelines",
				Action: func(c *cli.Context) error {
					for _, name := range utils.MapKeys(cfg.Pipelines) {
						fmt.Println(name)
					}

					return nil
				},
			},
			{
				Name:        "watchers",
				Description: "List watchers",
				Action: func(c *cli.Context) error {
					for _, name := range utils.MapKeys(cfg.Watchers) {
						fmt.Println(name)
					}

					return nil
				},
			},
		},
	}

	return cmd
}
