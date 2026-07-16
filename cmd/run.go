package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/scheduler"
	"github.com/taskctl/taskctl/task"
)

func newRunCommand() *cli.Command {
	var taskRunner *runner.TaskRunner
	cmd := &cli.Command{
		Name:      "Run",
		ArgsUsage: "Run (PIPELINE1 OR TASK1) [PIPELINE2 OR TASK2]... [flags] [-- TASKS_ARGS]",
		Usage:     "runs pipeline or task",
		UsageText: "taskctl Run pipeline1\n" +
			"taskctl pipeline1\n" +
			"taskctl task1",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-Run",
				Usage: "dry Run",
			},
			&cli.BoolFlag{
				Name:    "summary",
				Usage:   "show summary",
				Aliases: []string{"s"},
				Value:   true,
			},
		},
		Before: func(c *cli.Context) (err error) {
			cancelMu.Lock()
			c.Context, cancelFn = context.WithCancel(c.Context)
			cancelMu.Unlock()

			taskRunner, err = buildTaskRunner(c.Context, c)
			return err
		},
		After: func(c *cli.Context) error {
			return nil
		},
		Action: func(c *cli.Context) (err error) {
			if !c.Args().Present() {
				return fmt.Errorf("no target specified")
			}

			for _, v := range c.Args().Slice() {
				if v == "--" {
					break
				}

				if v == "pipeline" {
					continue
				}

				err = runTarget(v, c, taskRunner)
				if err != nil {
					return err
				}
			}
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:      "task",
				ArgsUsage: "task (TASK1) [TASK2]... [flags] [-- TASK_ARGS]",
				Usage:     "Run specified task(s)",
				Action: func(c *cli.Context) error {
					for _, v := range c.Args().Slice() {
						if v == "--" {
							break
						}

						t := cfg.Tasks[v]
						if t == nil {
							return fmt.Errorf("unknown task %s", v)
						}
						err := runTask(t, taskRunner)
						if err != nil {
							return err
						}
					}

					return nil
				},
			},
		},
	}

	return cmd
}

func runTarget(name string, c *cli.Context, taskRunner *runner.TaskRunner) (err error) {
	p := cfg.Pipelines[name]
	if p != nil {
		err = runPipeline(p, taskRunner, cfg.Summary || c.Bool("summary"))
		if err != nil {
			return fmt.Errorf("pipeline %s failed: %w", name, err)
		}
		return nil
	}

	t := cfg.Tasks[name]
	if t == nil {
		return fmt.Errorf("unknown task or pipeline %s", name)
	}
	err = runTask(t, taskRunner)
	if err != nil {
		return fmt.Errorf("task %s failed: %w", name, err)
	}

	return nil
}

func runPipeline(g *scheduler.ExecutionGraph, taskRunner *runner.TaskRunner, summary bool) error {
	sd := scheduler.NewScheduler(taskRunner)
	//go func() {
	//	<-cancelCh
	//	sd.Cancel()
	//}()

	err := sd.Schedule(g)
	if err != nil {
		return err
	}
	sd.Finish()

	_, _ = fmt.Fprint(os.Stdout, "\r\n")

	if summary {
		printSummary(g)
	}

	return nil
}

func runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t)
	if err != nil {
		return err
	}

	taskRunner.Finish()

	return nil
}

func taskArgs(c *cli.Context) []string {
	var runArgs []string
	var dash = -1
	for k, arg := range c.Args().Slice() {
		if arg == "--" {
			dash = k
		}
	}

	if dash >= 0 && dash != c.NArg()-1 {
		runArgs = c.Args().Slice()[dash+1:]
	}

	return runArgs
}

func printSummary(g *scheduler.ExecutionGraph) {
	var stages = make([]*scheduler.Stage, 0)
	for _, stage := range g.Nodes() {
		stages = append(stages, stage)
	}

	slices.SortFunc(stages, func(a, b *scheduler.Stage) int {
		return a.Start.Compare(b.Start)
	})

	tui.Println(os.Stdout, tui.StyleBold.Render("Summary:"))

	var log string
	for _, stage := range stages {
		switch stage.ReadStatus() {
		case scheduler.StatusDone:
			tui.Println(os.Stdout, tui.StyleSuccess.Render(fmt.Sprintf("- Stage %s was completed in %s", stage.Name, stage.Duration())))
		case scheduler.StatusSkipped:
			tui.Println(os.Stdout, tui.StyleSuccess.Render(fmt.Sprintf("- Stage %s was skipped", stage.Name)))
		case scheduler.StatusError:
			log = strings.TrimSpace(stage.Task.ErrorMessage())
			tui.Println(os.Stdout, tui.StyleError.Render(fmt.Sprintf("- Stage %s failed in %s", stage.Name, stage.Duration())))
			if log != "" {
				tui.Println(os.Stdout, tui.StyleError.Render(fmt.Sprintf("  > %s", log)))
			}
		case scheduler.StatusCanceled:
			tui.Println(os.Stdout, tui.StyleFaint.Render(fmt.Sprintf("- Stage %s was cancelled", stage.Name)))
		default:
			tui.Println(os.Stdout, tui.StyleError.Render(fmt.Sprintf("- Unexpected status %d for stage %s", stage.Status, stage.Name)))
		}
	}

	tui.Printf(os.Stdout, "%s: %s\n", tui.StyleBold.Render("Total duration"), tui.StyleSuccess.Render(g.Duration().String()))
}
