package cmd

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"text/template"

	"github.com/urfave/cli/v2"
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

			contexts := slices.Sorted(maps.Keys(cfg.Contexts))
			pipelines := slices.Sorted(maps.Keys(cfg.Pipelines))
			tasks := slices.Sorted(maps.Keys(cfg.Tasks))
			watchers := slices.Sorted(maps.Keys(cfg.Watchers))

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
					for _, name := range slices.Sorted(maps.Keys(cfg.Tasks)) {
						fmt.Println(name)
					}

					return nil
				},
			},
			{
				Name:        "pipelines",
				Description: "List pipelines",
				Action: func(c *cli.Context) error {
					for _, name := range slices.Sorted(maps.Keys(cfg.Pipelines)) {
						fmt.Println(name)
					}

					return nil
				},
			},
			{
				Name:        "watchers",
				Description: "List watchers",
				Action: func(c *cli.Context) error {
					for _, name := range slices.Sorted(maps.Keys(cfg.Watchers)) {
						fmt.Println(name)
					}

					return nil
				},
			},
		},
	}

	return cmd
}
