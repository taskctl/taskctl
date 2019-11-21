package cmd

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/scheduler"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
	"strings"
)

var quiet, raw bool

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "run (PIPELINE) [flags] [-- TASKS_ARGS]",
		Short:     "Run pipeline",
		ValidArgs: util.ListNames(cfg.Pipelines),
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}

			if err := cobra.OnlyValidArgs(cmd, args); err != nil {
				return err
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if raw && !debug {
				log.SetLevel(log.FatalLevel)
			}

			pipeline := pipelines[args[0]]

			var pipelineArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				pipelineArgs = args[cmd.ArgsLenAtDash():]
			}
			env := util.ConvertEnv(map[string]string{
				"ARGS": strings.Join(pipelineArgs, " "),
			})

			rr := scheduler.NewScheduler(pipeline, contexts, env, raw, quiet)
			go func() {
				select {
				case <-cancel:
					rr.Cancel()
					return
				}
			}()
			rr.Schedule()

			fmt.Println(aurora.Yellow("\r\nSummary:"))
			for _, stage := range pipeline.Nodes() {
				printSummary(stage)
			}

			fmt.Printf(aurora.Sprintf(aurora.Green("\r\nTotal duration: %s\r\n"), rr.End.Sub(rr.Start)))

			close(done)
		},
	}

	cmd.Flags().BoolVar(&raw, "raw-output", false, "raw output")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "disable tasks output")
	cmd.AddCommand(NewRunTaskCommand())

	return cmd
}

func printSummary(stage *scheduler.Stage) {
	switch stage.Task.ReadStatus() {
	case task.StatusDone:
		fmt.Printf(aurora.Sprintf(aurora.Green("- Stage %s done in %s\r\n"), stage.Name, stage.Task.Duration()))
	case task.StatusError:
		fmt.Printf(aurora.Sprintf(aurora.Red("- Stage %s failed in %s\r\n"), stage.Name, stage.Task.Duration()))
		fmt.Printf(aurora.Sprintf(aurora.Red("  Error: %s\r\n"), stage.Task.ReadLog()))
	case task.StatusCanceled:
		fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Stage %s is cancelled\r\n"), stage.Name))
	case task.StatusWaiting:
		fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Stage %s skipped\r\n"), stage.Name))
	default:
		log.Fatal(aurora.Sprintf(aurora.Red("- Unexpected status %d for task %s in stage\r\n"), stage.Task.Status, stage.Task.Name, stage.Name))
	}
}
