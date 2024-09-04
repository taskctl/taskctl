package cmdutils

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/logrusorgru/aurora"
)

// TUI helper section
// suggesion is a struct for TUI display
func rootAction() (err error) {
	// taskRunner, err := buildTaskRunner(c)

	// run pipeline or task

	// if err != nil {
	// 	return err
	// }

	// targets := c.Args().Slice()
	// if len(targets) > 0 {
	// 	for _, target := range targets {
	// 		if target == "--" {
	// 			break
	// 		}

	// 		err = runTarget(target, c, taskRunner)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// }

	// TUI Helper for when pipeline or task is empty
	// suggestions := buildSuggestions(cfg)
	// targetSelect := promptui.Select{
	// 	Label:        "Select task to run",
	// 	Items:        suggestions,
	// 	Size:         15,
	// 	CursorPos:    0,
	// 	IsVimMode:    false,
	// 	HideHelp:     false,
	// 	HideSelected: false,
	// 	Templates: &promptui.SelectTemplates{
	// 		Active:   fmt.Sprintf("%s {{ .DisplayName | underline }}", promptui.IconSelect),
	// 		Inactive: "  {{ .DisplayName }}",
	// 		Selected: fmt.Sprintf(`{{ "%s" | green }} {{ .DisplayName | faint }}`, promptui.IconGood),
	// 	},
	// 	Keys: nil,
	// 	Searcher: func(input string, index int) bool {
	// 		return strings.Contains(suggestions[index].DisplayName, input)
	// 	},
	// 	StartInSearchMode: true,
	// }

	// fmt.Println("Please use `Ctrl-C` to exit this program.")
	// index, _, err := targetSelect.Run()
	// if err != nil {
	// 	return err
	// }

	// selection := suggestions[index]
	// if selection.IsTask {
	// 	return runTask(cfg.Tasks[selection.Target], taskRunner)
	// }

	// return runPipeline(cfg.Pipelines[selection.Target], taskRunner, cfg.Summary || c.Bool("summary"))

	return nil
}

type suggestion struct {
	Target, DisplayName string
	IsTask              bool
}

func buildSuggestions(cfg *config.Config) []suggestion {
	if cfg == nil {
		return nil
	}

	suggestions := make([]suggestion, 0)

	for _, v := range utils.MapKeys(cfg.Pipelines) {
		suggestions = append(suggestions, suggestion{
			Target:      v,
			DisplayName: fmt.Sprintf("%s - %s", v, aurora.Gray(12, "pipeline").String()),
		})
	}

	for k, v := range cfg.Tasks {
		desc := "task"
		if v.Description != "" {
			desc = v.Description
		}
		suggestions = append(suggestions, suggestion{
			Target:      k,
			DisplayName: fmt.Sprintf("%s - %s", k, aurora.Gray(12, desc).String()),
			IsTask:      true,
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[j].Target > suggestions[i].Target
	})

	return suggestions
}

// printSummary is a TUI helper
func PrintSummary(g *scheduler.ExecutionGraph, chanOut io.Writer) {
	var stages = make([]*scheduler.Stage, 0)
	for _, stage := range g.Nodes() {
		stages = append(stages, stage)
	}

	sort.Slice(stages, func(i, j int) bool {
		return stages[j].Start.Nanosecond() > stages[i].Start.Nanosecond()
	})

	fmt.Fprintln(chanOut, aurora.Bold("Summary:").String())

	var log string
	for _, stage := range stages {
		switch stage.ReadStatus() {
		case scheduler.StatusDone:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Green("- Stage %s was completed in %s"), stage.Name, stage.Duration()))
		case scheduler.StatusSkipped:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Green("- Stage %s was skipped"), stage.Name))
		case scheduler.StatusError:
			log = strings.TrimSpace(stage.Task.ErrorMessage())
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Red("- Stage %s failed in %s"), stage.Name, stage.Duration()))
			if log != "" {
				fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Red("  > %s"), log))
			}
		case scheduler.StatusCanceled:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Gray(12, "- Stage %s was cancelled"), stage.Name))
		default:
			fmt.Fprintln(chanOut, aurora.Sprintf(aurora.Red("- Unexpected status %d for stage %s"), stage.Status, stage.Name))
		}
	}

	fmt.Fprintln(chanOut, aurora.Sprintf("%s: %s", aurora.Bold("Total duration"), aurora.Green(g.Duration())))
}
