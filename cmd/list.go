package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/collections"
	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/schema"
	"github.com/taskctl/taskctl/internal/tui"
)

func newListCommand(cfg *config.Config) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "lists contexts, pipelines, tasks and watchers",
		Long:  "Lists everything declared in the config. With --output json, emits a schema-versioned discovery document intended for machine/agent consumption.",
		Example: "  taskctl list\n" +
			"  taskctl list --output json",
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

			renderList(os.Stdout, cfg, contexts, pipelines, tasks, watchers)
			return nil
		},
	}

	// tasks/pipelines/watchers share the human path (one name per line); they
	// differ only in the typed JSON document they emit.
	subs := []struct {
		use, short, long, example string
		names                     func() []string
		jsonDoc                   func() any
	}{
		{"tasks", "list tasks", "Lists task names one per line; with --output json, a schema-versioned array of task summaries.", "  taskctl list tasks", sortedKeys(cfg.Tasks), func() any {
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
		{"pipelines", "list pipelines", "Lists pipeline names one per line; with --output json, a schema-versioned array of pipeline summaries with their stages.", "  taskctl list pipelines", sortedKeys(cfg.Pipelines), func() any {
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
		{"watchers", "list watchers", "Lists watcher names one per line; with --output json, a schema-versioned list of watcher names.", "  taskctl list watchers", sortedKeys(cfg.Watchers), func() any {
			return struct {
				SchemaVersion int      `json:"schema_version"`
				Watchers      []string `json:"watchers"`
			}{1, collections.OrEmpty(slices.Sorted(maps.Keys(cfg.Watchers)))}
		}},
	}

	for _, s := range subs {
		listCmd.AddCommand(&cobra.Command{
			Use:     s.use,
			Short:   s.short,
			Long:    s.long,
			Example: s.example,
			Args:    cobra.NoArgs,
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

// renderList writes the styled human listing: a bold section header per
// non-empty group, task names aligned against their faint descriptions.
func renderList(w io.Writer, cfg *config.Config, contexts, pipelines, tasks, watchers []string) {
	first := true
	header := func(title string) {
		if !first {
			tui.Println(w, "")
		}
		first = false
		tui.Println(w, tui.StyleBold.Render(strings.ToUpper(title)))
	}

	if len(pipelines) > 0 {
		header("Pipelines")
		for _, name := range pipelines {
			tui.Printf(w, "  %s\n", name)
		}
	}

	if len(tasks) > 0 {
		header("Tasks")
		width := 0
		for _, name := range tasks {
			width = max(width, lipgloss.Width(name))
		}
		for _, name := range tasks {
			desc := cfg.Tasks[name].Description
			if desc == "" {
				tui.Printf(w, "  %s\n", name)
				continue
			}
			pad := strings.Repeat(" ", width-lipgloss.Width(name)+2)
			tui.Printf(w, "  %s%s%s\n", name, pad, tui.StyleFaint.Render(desc))
		}
	}

	if len(contexts) > 0 {
		header("Contexts")
		for _, name := range contexts {
			tui.Printf(w, "  %s\n", name)
		}
	}

	if len(watchers) > 0 {
		header("Watchers")
		for _, name := range watchers {
			tui.Printf(w, "  %s\n", name)
		}
	}

	if first {
		tui.Println(w, tui.StyleFaint.Render("No tasks or pipelines found."))
	}
}
