package cmd

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/task"
	"strings"
)

var quiet, raw bool

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [pipeline]",
		Short: "Run pipeline",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var pname = args[0]
			pipeline, ok := pipelines[pname]
			if !ok {
				logrus.Fatalf("unknown pipeline %s", pname)
			}

			var pipelineArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				pipelineArgs = args[cmd.ArgsLenAtDash():]
			}
			env := config.ConvertEnv(map[string]string{
				"ARGS": strings.Join(pipelineArgs, " "),
			})

			rr := runner.NewScheduler(pipeline, contexts, env, raw, quiet)
			go func() {
				select {
				case <-cancel:
					rr.Cancel()
					return
				}
			}()
			rr.Schedule()

			fmt.Println(aurora.Yellow("\r\nSummary:"))
			for _, t := range pipeline.Nodes() {
				printSummary(t)
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

func printSummary(t *task.Task) {
	switch t.ReadStatus() {
	case task.STATUS_DONE:
		fmt.Printf(aurora.Sprintf(aurora.Green("- Task %s done in %s\r\n"), t.Name, t.Duration()))
	case task.STATUS_ERROR:
		fmt.Printf(aurora.Sprintf(aurora.Red("- Task %s failed in %s\r\n"), t.Name, t.Duration()))
		fmt.Printf(aurora.Sprintf(aurora.Red("  Error: %s\r\n"), t.ReadLog()))
	case task.STATUS_CANCELED:
		fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Task %s is cancelled\r\n"), t.Name))
	case task.STATUS_WAITING:
		fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Task %s skipped\r\n"), t.Name))
	default:
		logrus.Fatal(aurora.Sprintf(aurora.Red("- Unexpected status %d for task %s\r\n"), t.Status, t.Name))
	}
}
