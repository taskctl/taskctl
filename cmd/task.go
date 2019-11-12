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
	runCommand.AddCommand(taskRunCommand)
}

var taskRunCommand = &cobra.Command{
	Use: "task [task]",
	Short: "Run task",
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var tname = args[0]
		t, ok := tasks[tname]
		if !ok {
			logrus.Fatalf("unknown task %s", tname)
		}

		rr := runner.NewRunner(nil, contexts)
		rr.RunTask(t)

		fmt.Println(aurora.Yellow("\r\nSummary:"))
		switch t.ReadStatus() {
		case task.STATUS_DONE:
			fmt.Printf(aurora.Sprintf(aurora.Green("- Task %s done in %s\r\n"), t.Name, t.Duration()))
		case task.STATUS_ERROR:
			fmt.Printf(aurora.Sprintf(aurora.Red("- Task %s failed in %s\r\n"), t.Name, t.Duration()))
		default:
			fmt.Printf(aurora.Sprintf(aurora.Red("- Unexpected status %d for task %s\r\n"), t.Status, t.Name))
		}

		close(done)
	},
}
