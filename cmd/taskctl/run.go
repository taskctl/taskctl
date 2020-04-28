package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"sort"
	"strings"

	"github.com/taskctl/taskctl/pkg/output"

	"github.com/logrusorgru/aurora"
	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/scheduler"
	"github.com/taskctl/taskctl/pkg/task"
	"github.com/taskctl/taskctl/pkg/util"
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
				return fmt.Errorf("no Target specified")
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
			&cli.Command{
				Name:        "task",
				Usage:       "task (TASK1) [TASK2]... [flags] [-- TASK_ARGS]",
				Description: "run specified task(s)",
				Action: func(c *cli.Context) error {
					for _, v := range c.Args().Slice() {
						if v == "--" {
							break
						}

						t, ok := tasks[v]
						if !ok {
							return fmt.Errorf("unknown task or pipeline %s", v)
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
	var dash = -1
	for k, arg := range c.Args().Slice() {
		if arg == "--" {
			dash = k
		}
	}

	var runArgs []string
	if dash >= 0 {
		runArgs = c.Args().Slice()[dash:]
	}

	env := util.ConvertEnv(map[string]string{
		"ARGS": strings.Join(runArgs, " "),
	})

	taskRunner, err := runner.NewTaskRunner(contexts, env, c.String("output"), c.Bool("dry-run"), cfg.Variables)
	if err != nil {
		return nil, err
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
	err := taskRunner.Run(t)
	if err != nil {
		return err
	}
	taskRunner.Finish()

	return nil
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
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Green("- Stage %s done in %s"), stage.Name, stage.Duration()))
		case pipeline.StatusError:
			log = strings.TrimSpace(stage.Task.Error())
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Red("- Stage %s failed in %s"), stage.Name, stage.Duration()))
			if log != "" {
				fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Red("  > %s"), log))
			}
		case pipeline.StatusCanceled:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Gray(12, "- Stage %s is cancelled"), stage.Name))
		case pipeline.StatusWaiting:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Gray(12, "- Stage %s skipped"), stage.Name))
		default:
			fmt.Fprintln(output.Stdout, aurora.Sprintf(aurora.Red("- Unexpected status %d for stage %s"), stage.Status, stage.Name))
		}
	}
}
