package cmd

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/task"
)

func init() {
	rootCmd.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	Use: "run [pipeline]",
	Short: "Run pipeline",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var pname = args[0]
		pipeline, ok := pipelines[pname]
		if !ok {
			logrus.Fatalf("unknown pipeline %s", pname)
		}

		rr := runner.NewRunner(pipeline, contexts)
		go func() {
			select {
			case <-cancel:
				rr.Cancel()
				return
			}
		}()
		rr.Run()

		fmt.Println(aurora.Yellow("\r\nSummary:"))
		for _, t := range pipeline.Nodes() {
			switch t.ReadStatus() {
			case task.STATUS_DONE:
				fmt.Printf(aurora.Sprintf(aurora.Green("- Task %s done in %s\r\n"), t.Name, t.Duration()))
			case task.STATUS_ERROR:
				fmt.Printf(aurora.Sprintf(aurora.Red("- Task %s failed in %s\r\n"), t.Name, t.Duration()))
			case task.STATUS_CANCELED:
				fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Task %s is cancelled\r\n"), t.Name))
			case task.STATUS_WAITING:
				fmt.Printf(aurora.Sprintf(aurora.Gray(12, "- Task %s skipped\r\n"), t.Name))
			default:
				logrus.Fatal(aurora.Sprintf(aurora.Red("- Unexpected status %d for task %s\r\n"), t.Status, t.Name))
			}
		}

		fmt.Printf(aurora.Sprintf(aurora.Green("Total duration: %s\r\n"), rr.End.Sub(rr.Start)))

		close(done)
	},
}
