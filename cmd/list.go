package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/template"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/schema"
	"github.com/taskctl/taskctl/output"
	"github.com/taskctl/taskctl/utils"
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
			contexts := utils.MapKeys(cfg.Contexts)
			pipelines := utils.MapKeys(cfg.Pipelines)
			tasks := utils.MapKeys(cfg.Tasks)
			watchers := utils.MapKeys(cfg.Watchers)

			sort.Strings(contexts)
			sort.Strings(pipelines)
			sort.Strings(tasks)
			sort.Strings(watchers)

			if cfg.Output == output.FormatJSON {
				return encodeListJSON(contexts, pipelines, tasks, watchers)
			}

			t := template.Must(template.New("list").Parse(listTmpl))

			return t.Execute(os.Stdout, struct {
				Contexts, Pipelines, Tasks, Watchers []string
			}{
				Contexts:  contexts,
				Pipelines: pipelines,
				Tasks:     tasks,
				Watchers:  watchers,
			})
		},
		Subcommands: []*cli.Command{
			{
				Name:        "tasks",
				Description: "List tasks",
				Action: func(c *cli.Context) error {
					names := utils.MapKeys(cfg.Tasks)

					if cfg.Output == output.FormatJSON {
						sort.Strings(names)

						summaries := make([]schema.TaskSummary, 0, len(names))
						for _, name := range names {
							summaries = append(summaries, schema.NewTaskSummary(cfg.Tasks[name]))
						}

						return json.NewEncoder(os.Stdout).Encode(struct {
							SchemaVersion int                  `json:"schema_version"`
							Tasks         []schema.TaskSummary `json:"tasks"`
						}{1, summaries})
					}

					for _, name := range names {
						fmt.Println(name)
					}

					return nil
				},
			},
			{
				Name:        "pipelines",
				Description: "List pipelines",
				Action: func(c *cli.Context) error {
					names := utils.MapKeys(cfg.Pipelines)

					if cfg.Output == output.FormatJSON {
						sort.Strings(names)

						summaries := make([]schema.PipelineSummary, 0, len(names))
						for _, name := range names {
							summaries = append(summaries, schema.NewPipelineSummary(name, cfg.Pipelines[name]))
						}

						return json.NewEncoder(os.Stdout).Encode(struct {
							SchemaVersion int                      `json:"schema_version"`
							Pipelines     []schema.PipelineSummary `json:"pipelines"`
						}{1, summaries})
					}

					for _, name := range names {
						fmt.Println(name)
					}

					return nil
				},
			},
			{
				Name:        "watchers",
				Description: "List watchers",
				Action: func(c *cli.Context) error {
					names := utils.MapKeys(cfg.Watchers)

					if cfg.Output == output.FormatJSON {
						sort.Strings(names)

						return json.NewEncoder(os.Stdout).Encode(struct {
							SchemaVersion int      `json:"schema_version"`
							Watchers      []string `json:"watchers"`
						}{1, utils.OrEmpty(names)})
					}

					for _, name := range names {
						fmt.Println(name)
					}

					return nil
				},
			},
		},
	}

	return cmd
}

// encodeListJSON writes the schema_version-tagged discovery document for
// `taskctl --output json list`. All four keys are always present, even when
// empty, so the sorted name slices are used to build non-nil summary slices
// regardless of length.
func encodeListJSON(contexts, pipelineNames, taskNames, watchers []string) error {
	taskSummaries := make([]schema.TaskSummary, 0, len(taskNames))
	for _, name := range taskNames {
		taskSummaries = append(taskSummaries, schema.NewTaskSummary(cfg.Tasks[name]))
	}

	pipelineSummaries := make([]schema.PipelineSummary, 0, len(pipelineNames))
	for _, name := range pipelineNames {
		pipelineSummaries = append(pipelineSummaries, schema.NewPipelineSummary(name, cfg.Pipelines[name]))
	}

	resp := schema.ListResponse{
		SchemaVersion: 1,
		Tasks:         taskSummaries,
		Pipelines:     pipelineSummaries,
		Contexts:      utils.OrEmpty(contexts),
		Watchers:      utils.OrEmpty(watchers),
	}

	return json.NewEncoder(os.Stdout).Encode(resp)
}
