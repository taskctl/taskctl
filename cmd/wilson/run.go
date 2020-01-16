package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/pipeline"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/scheduler"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
	"strings"
)

var quiet, raw bool

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run (PIPELINE1 OR TASK1) [PIPELINE2 OR TASK2]... [flags] [-- TASKS_ARGS]",
		Short: "Run pipeline or task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = loadConfig()
			if err != nil {
				return err
			}

			if raw && !debug {
				log.SetLevel(log.FatalLevel)
			}

			list := make([]string, 0)
			if cmd.ArgsLenAtDash() > 0 {
				list = args[:cmd.ArgsLenAtDash()]
			}
			for _, v := range list {
				pipeline, ok := pipelines[v]
				if ok {
					err = runPipeline(pipeline, cmd, args)
				} else {
					t, ok := tasks[v]
					if !ok {
						return fmt.Errorf("unknown task or pipeline %s", v)
					}
					err = runTask(t, cmd, args)
				}
				if err != nil {
					break
				}
			}
			close(done)

			return err
		},
	}

	cmd.Flags().BoolVar(&raw, "raw-output", false, "raw output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "disable tasks output")
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

			for _, v := range args {
				t, ok := tasks[v]
				if !ok {
					return fmt.Errorf("unknown task %s", v)
				}

				err = runTask(t, cmd, args)
				if err != nil {
					break
				}
			}
			close(done)
			return err
		},
	}
}

func runPipeline(pipeline *pipeline.Pipeline, cmd *cobra.Command, args []string) error {
	var pipelineArgs []string
	if al := cmd.ArgsLenAtDash(); al > 0 {
		pipelineArgs = args[cmd.ArgsLenAtDash():]
	}
	env := util.ConvertEnv(map[string]string{
		"ARGS": strings.Join(pipelineArgs, " "),
	})

	rr := scheduler.NewScheduler(contexts, env, raw, quiet)
	go func() {
		select {
		case <-cancel:
			rr.Cancel()
			return
		}
	}()

	cmd.SilenceUsage = true
	err := rr.Schedule(pipeline)
	if err != nil {
		return err
	}
	rr.DownContexts()

	fmt.Println(aurora.Yellow("\r\nSummary:"))
	printSummary(pipeline)

	fmt.Printf(aurora.Sprintf(aurora.Green("\r\nTotal duration: %s\r\n"), rr.End.Sub(rr.Start)))

	return nil
}

func runTask(t *task.Task, cmd *cobra.Command, args []string) error {
	var taskArgs []string
	if al := cmd.ArgsLenAtDash(); al > 0 {
		taskArgs = args[cmd.ArgsLenAtDash():]
	}
	env := util.ConvertEnv(map[string]string{
		"ARGS": strings.Join(taskArgs, " "),
	})

	cmd.SilenceUsage = true
	tr := runner.NewTaskRunner(contexts, env, true, quiet)
	err := tr.Run(t)
	if err != nil {
		return err
	}
	tr.DownContexts()

	return nil
}

func printSummary(p *pipeline.Pipeline) {
	// todo: order by start time
	for _, stage := range p.Nodes() {
		switch stage.ReadStatus() {
		case pipeline.StatusDone:
			fmt.Printf(aurora.Sprintf(aurora.Green("- Stage %s done in %s\r\n"), stage.Name, stage.Duration()))
		case pipeline.StatusError:
			fmt.Printf(aurora.Sprintf(aurora.Red("- Stage %s failed in %s\r\n"), stage.Name, stage.Duration()))
			fmt.Printf(aurora.Sprintf(aurora.Red("  Error: %s\r\n"), stage.Task.ReadLog()))
		case pipeline.StatusCanceled:
			fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Stage %s is cancelled\r\n"), stage.Name))
		case pipeline.StatusWaiting:
			fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Stage %s skipped\r\n"), stage.Name))
		default:
			log.Errorf(aurora.Sprintf(aurora.Red("- Unexpected status %d for stage %s\r\n"), stage.Status, stage.Name))
		}
	}
}
