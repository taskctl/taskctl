package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/taskctl/taskctl/pkg/config"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/pkg/output"

	"github.com/logrusorgru/aurora"

	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/scheduler"
	"github.com/taskctl/taskctl/pkg/task"
)

func NewRunCommand() *cli.Command {
	var taskRunner *runner.TaskRunner
	cmd := &cli.Command{
		Name:      "run",
		ArgsUsage: "run (PIPELINE1 OR TASK1) [PIPELINE2 OR TASK2]... [flags] [-- TASKS_ARGS]",
		Usage:     "runs pipeline or task",
		UsageText: "taskctl run pipeline1\n" +
			"taskctl pipeline1\n" +
			"taskctl task1",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "dry run",
			},
			&cli.BoolFlag{
				Name:    "summary",
				Usage:   "show summary",
				Aliases: []string{"s"},
				Value:   true,
			},
		},
		Before: func(c *cli.Context) (err error) {
			taskRunner, err = buildTaskRunner(c)
			if err != nil {
				return err
			}

			return nil
		},
		After: func(c *cli.Context) error {
			close(done)
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
				Usage:     "run specified task(s)",
				Action: func(c *cli.Context) error {
					for _, v := range c.Args().Slice() {
						if v == "--" {
							break
						}

						t, ok := tasks[v]
						if !ok {
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

func buildTaskRunner(c *cli.Context) (*runner.TaskRunner, error) {
	variables := cfg.Variables.With("args", strings.Join(taskArgs(c), " "))
	taskRunner, err := runner.NewTaskRunner(contexts, c.String("output"), variables)
	if err != nil {
		return nil, err
	}

	if c.Bool("dry-run") {
		taskRunner.DryRun()
	}

	return taskRunner, nil
}

func runTarget(name string, c *cli.Context, taskRunner *runner.TaskRunner) (err error) {
	p, ok := pipelines[name]
	if ok {
		err = runPipeline(p, taskRunner, c.Bool("summary"))
		if err != nil {
			return err
		}
		return nil
	}

	t, ok := tasks[name]
	if !ok {
		return fmt.Errorf("unknown task or pipeline %s", name)
	}
	err = runTask(t, taskRunner)
	return err
}

func runPipeline(pipeline *pipeline.Pipeline, taskRunner *runner.TaskRunner, summary bool) error {
	sd := scheduler.NewScheduler(taskRunner)
	go func() {
		<-cancel
		sd.Cancel()
	}()

	err := sd.Schedule(pipeline)
	if err != nil {
		return err
	}
	sd.Finish()

	if summary {
		printSummary(pipeline)
	}

	fmt.Fprintln(output.Stdout, aurora.Sprintf("\r\n%s: %s", aurora.Bold("Total duration"), aurora.Green(sd.End.Sub(sd.Start))))

	return nil
}

func runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t, config.Set{}, config.Set{})
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

func printSummary(p *pipeline.Pipeline) {
	var stages = make([]*pipeline.Stage, 0)
	for _, stage := range p.Nodes() {
		stages = append(stages, stage)
	}

	sort.Slice(stages, func(i, j int) bool {
		return stages[j].Start.Nanosecond() > stages[i].Start.Nanosecond()
	})

	fmt.Fprintln(output.Stdout, aurora.Bold("\r\nSummary:").String())

	var log string
	for _, stage := range stages {
		switch stage.ReadStatus() {
		case pipeline.StatusDone:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Green("- Stage %s was completed in %s"), stage.Name, stage.Duration()))
		case pipeline.StatusSkipped:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Green("- Stage %s was skipped"), stage.Name))
		case pipeline.StatusError:
			log = strings.TrimSpace(stage.Task.ErrorMessage())
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Red("- Stage %s failed in %s"), stage.Name, stage.Duration()))
			if log != "" {
				fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Red("  > %s"), log))
			}
		case pipeline.StatusCanceled:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Gray(12, "- Stage %s was cancelled"), stage.Name))
		default:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Red("- Unexpected status %d for stage %s"), stage.Status, stage.Name))
		}
	}
}
