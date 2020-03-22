package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/pkg/output"

	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/scheduler"
	"github.com/taskctl/taskctl/pkg/task"
	"github.com/taskctl/taskctl/pkg/util"
)

var summary, dryRun bool

func NewRunCommand() *cobra.Command {
	cfg := config.Get()
	cmd := &cobra.Command{
		Use:   "run (PIPELINE1 OR TASK1) [PIPELINE2 OR TASK2]... [flags] [-- TASKS_ARGS]",
		Short: "Run pipeline or task",
		Args:  cobra.MinimumNArgs(1),
		Example: "  taskctl run pipeline1\n" +
			"  taskctl pipeline1\n" +
			"  taskctl task1",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = loadConfig()
			if err != nil {
				return err
			}

			if !debug {
				logrus.SetLevel(logrus.FatalLevel)
			}

			targets := make([]string, 0)
			if cmd.ArgsLenAtDash() > 0 {
				targets = args[:cmd.ArgsLenAtDash()]
			} else {
				targets = args
			}

			var runArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				runArgs = args[cmd.ArgsLenAtDash():]
			}
			env := util.ConvertEnv(map[string]string{
				"ARGS": strings.Join(runArgs, " "),
			})
			rn, err := runner.NewTaskRunner(contexts, env, oflavor, dryRun)
			if err != nil {
				return err
			}

			for _, v := range targets {
				p, ok := pipelines[v]
				if ok {
					err = runPipeline(p, cmd, rn)
				} else {
					t, ok := tasks[v]
					if !ok {
						return fmt.Errorf("unknown task or pipeline %s", v)
					}
					err = runTask(t, cmd, rn)
				}
				if err != nil {
					break
				}
			}
			close(done)

			return err
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", cfg.DryRun, "dry run")
	cmd.Flags().BoolVarP(&summary, "summary", "s", true, "show summary")
	cmd.AddCommand(NewRunTaskCommand())

	return cmd
}

func NewRunTaskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "task (TASK1) [TASK2]... [flags] [-- TASK_ARGS]",
		Short: "Run task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = loadConfig()
			if err != nil {
				return err
			}

			var runArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				runArgs = args[cmd.ArgsLenAtDash():]
			}
			env := util.ConvertEnv(map[string]string{
				"ARGS": strings.Join(runArgs, " "),
			})
			rn, err := runner.NewTaskRunner(contexts, env, oflavor, dryRun)

			for _, v := range args {
				t, ok := tasks[v]
				if !ok {
					return fmt.Errorf("unknown task %s", v)
				}

				err = runTask(t, cmd, rn)
				if err != nil {
					break
				}
			}
			close(done)
			return err
		},
	}
}

func runPipeline(pipeline *pipeline.Pipeline, cmd *cobra.Command, rn *runner.TaskRunner) error {
	sd := scheduler.NewScheduler(rn)
	go func() {
		select {
		case <-cancel:
			sd.Cancel()
			return
		}
	}()

	cmd.SilenceUsage = true
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

func runTask(t *task.Task, cmd *cobra.Command, rn *runner.TaskRunner) error {
	cmd.SilenceUsage = true

	err := rn.Run(t)
	if err != nil {
		return err
	}
	rn.Finish()

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
