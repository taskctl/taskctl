package cmd

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
)

var quiet, raw bool

func init() {
	runCommand.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "silence output")
	runCommand.PersistentFlags().BoolVar(&raw, "raw-output", false, "raw output")
	rootCmd.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	Use:   "run [pipeline]",
	Short: "Run pipeline",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var pname = args[0]
		pipeline, ok := pipelines[pname]
		if !ok {
			logrus.Fatalf("unknown pipeline %s", pname)
		}

		rr := runner.NewRunner(pipeline, contexts, raw, quiet)
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
