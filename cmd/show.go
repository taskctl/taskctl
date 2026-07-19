package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/schema"
	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/task"
)

func newShowCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:               "show TASK_OR_PIPELINE",
		Short:             "shows a task's or pipeline's details",
		GroupID:           groupInspect,
		Args:              exactArgs(1, "show requires exactly one task or pipeline name"),
		ValidArgsFunction: targetCompletion(cfg),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			if t := cfg.Tasks[name]; t != nil {
				if cfg.Output == output.FormatJSON {
					// Mirror the compiler's precedence: task variables override config ones.
					vars := cfg.Variables.Merge(t.Variables).Map()
					return json.NewEncoder(os.Stdout).Encode(struct {
						SchemaVersion int               `json:"schema_version"`
						Task          schema.TaskDetail `json:"task"`
					}{1, schema.NewTaskDetail(t, vars)})
				}
				renderTask(os.Stdout, t)
				return nil
			}

			if g := cfg.Pipelines[name]; g != nil {
				detail := schema.NewPipelineDetail(name, g)
				if cfg.Output == output.FormatJSON {
					return json.NewEncoder(os.Stdout).Encode(struct {
						SchemaVersion int                   `json:"schema_version"`
						Pipeline      schema.PipelineDetail `json:"pipeline"`
					}{1, detail})
				}
				renderPipeline(os.Stdout, detail)
				return nil
			}

			return fmt.Errorf("unknown task or pipeline %q", name)
		},
	}
}

func renderTask(w io.Writer, t *task.Task) {
	title := tui.StyleBold.Render(t.Name)
	if t.Description != "" {
		title += "  " + tui.StyleFaint.Render(t.Description)
	}
	tui.Println(w, "")
	tui.Println(w, title)
	tui.Println(w, "")

	row := func(k, v string) {
		tui.Printf(w, "  %s  %s\n", tui.StyleFaint.Render(fmt.Sprintf("%-13s", k)), v)
	}

	ctx := t.Context
	if ctx == "" {
		ctx = tui.StyleFaint.Render("(default)")
	}
	row("Context", ctx)

	tui.Printf(w, "  %s\n", tui.StyleFaint.Render("Commands"))
	for _, c := range t.Commands {
		tui.Printf(w, "    %s\n", c)
	}

	if t.Dir != "" {
		row("Dir", t.Dir)
	}
	if t.Timeout != nil {
		row("Timeout", t.Timeout.String())
	}
	row("Allow failure", fmt.Sprintf("%t", t.AllowFailure))
}

func renderPipeline(w io.Writer, detail schema.PipelineDetail) {
	tui.Println(w, "")
	tui.Println(w, tui.StyleBold.Render(detail.Name)+"  "+tui.StyleFaint.Render("(pipeline)"))
	tui.Println(w, "")

	for _, s := range detail.Stages {
		line := "  " + s.Name
		if len(s.DependsOn) > 0 {
			line += "  " + tui.StyleFaint.Render("depends on: "+strings.Join(s.DependsOn, ", "))
		}
		tui.Println(w, line)
	}
}
