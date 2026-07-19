package cmd

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/collections"
	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/schema"
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

func newListCommand(cfg *config.Config) *cobra.Command {
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "lists contexts, pipelines, tasks and watchers",
		GroupID: groupInspect,
		Args:    cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			contexts := slices.Sorted(maps.Keys(cfg.Contexts))
			pipelines := slices.Sorted(maps.Keys(cfg.Pipelines))
			tasks := slices.Sorted(maps.Keys(cfg.Tasks))
			watchers := slices.Sorted(maps.Keys(cfg.Watchers))

			if cfg.Output == output.FormatJSON {
				return encodeListJSON(cfg, contexts, pipelines, tasks, watchers)
			}

			t := template.Must(template.New("list").Parse(listTmpl))
			return t.Execute(os.Stdout, struct {
				Contexts, Pipelines, Tasks, Watchers []string
			}{contexts, pipelines, tasks, watchers})
		},
	}

	// tasks/pipelines/watchers share the human path (one name per line); they
	// differ only in the typed JSON document they emit.
	subs := []struct {
		use, short string
		names      func() []string
		jsonDoc    func() any
	}{
		{"tasks", "List tasks", sortedKeys(cfg.Tasks), func() any {
			names := slices.Sorted(maps.Keys(cfg.Tasks))
			summaries := make([]schema.TaskSummary, 0, len(names))
			for _, name := range names {
				summaries = append(summaries, schema.NewTaskSummary(cfg.Tasks[name]))
			}
			return struct {
				SchemaVersion int                  `json:"schema_version"`
				Tasks         []schema.TaskSummary `json:"tasks"`
			}{1, summaries}
		}},
		{"pipelines", "List pipelines", sortedKeys(cfg.Pipelines), func() any {
			names := slices.Sorted(maps.Keys(cfg.Pipelines))
			summaries := make([]schema.PipelineSummary, 0, len(names))
			for _, name := range names {
				summaries = append(summaries, schema.NewPipelineSummary(name, cfg.Pipelines[name]))
			}
			return struct {
				SchemaVersion int                      `json:"schema_version"`
				Pipelines     []schema.PipelineSummary `json:"pipelines"`
			}{1, summaries}
		}},
		{"watchers", "List watchers", sortedKeys(cfg.Watchers), func() any {
			return struct {
				SchemaVersion int      `json:"schema_version"`
				Watchers      []string `json:"watchers"`
			}{1, collections.OrEmpty(slices.Sorted(maps.Keys(cfg.Watchers)))}
		}},
	}

	for _, s := range subs {
		listCmd.AddCommand(&cobra.Command{
			Use:   s.use,
			Short: s.short,
			Args:  cobra.NoArgs,
			RunE: func(*cobra.Command, []string) error {
				if cfg.Output == output.FormatJSON {
					return json.NewEncoder(os.Stdout).Encode(s.jsonDoc())
				}
				for _, n := range s.names() {
					fmt.Println(n)
				}
				return nil
			},
		})
	}

	return listCmd
}

func sortedKeys[V any](m map[string]V) func() []string {
	return func() []string { return slices.Sorted(maps.Keys(m)) }
}

// encodeListJSON writes the schema_version-tagged discovery document for
// `taskctl --output json list`. All four keys are always present, even when
// empty, so the sorted name slices are used to build non-nil summary slices
// regardless of length.
func encodeListJSON(cfg *config.Config, contexts, pipelineNames, taskNames, watchers []string) error {
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
		Contexts:      collections.OrEmpty(contexts),
		Watchers:      collections.OrEmpty(watchers),
	}

	return json.NewEncoder(os.Stdout).Encode(resp)
}
