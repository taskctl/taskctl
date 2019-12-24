package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/scheduler"
	"github.com/trntv/wilson/pkg/util"
	"strings"
)

var quiet, raw bool

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run (PIPELINE) [flags] [-- TASKS_ARGS]",
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

			pipeline, ok := pipelines[args[0]]
			if ok {
				runPipeline(pipeline, cmd, args)
			} else {
				t, ok := tasks[args[0]]
				if !ok {
					return fmt.Errorf("unknown task %s", args[0])
				}

				runTask(t, cmd, args)
			}

			close(done)

			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw-output", false, "raw output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "disable tasks output")
	cmd.AddCommand(NewRunTaskCommand())

	return cmd
}

func runPipeline(pipeline *scheduler.Pipeline, cmd *cobra.Command, args []string) {
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
	rr.Schedule(pipeline)
	rr.DownContexts()

	fmt.Println(aurora.Yellow("\r\nSummary:"))
	printSummary(pipeline)

	fmt.Printf(aurora.Sprintf(aurora.Green("\r\nTotal duration: %s\r\n"), rr.End.Sub(rr.Start)))
}

func printSummary(pipeline *scheduler.Pipeline) {
	// todo: order by start time
	for _, stage := range pipeline.Nodes() {
		switch stage.ReadStatus() {
		case scheduler.StatusDone:
			fmt.Printf(aurora.Sprintf(aurora.Green("- Stage %s done in %s\r\n"), stage.Name, stage.Duration()))
		case scheduler.StatusError:
			fmt.Printf(aurora.Sprintf(aurora.Red("- Stage %s failed in %s\r\n"), stage.Name, stage.Duration()))
			fmt.Printf(aurora.Sprintf(aurora.Red("  Error: %s\r\n"), stage.Task.ReadLog()))
		case scheduler.StatusCanceled:
			fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Stage %s is cancelled\r\n"), stage.Name))
		case scheduler.StatusWaiting:
			fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Stage %s skipped\r\n"), stage.Name))
		default:
			log.Errorf(aurora.Sprintf(aurora.Red("- Unexpected status %d for stage %s\r\n"), stage.Status, stage.Name))
		}
	}
}
