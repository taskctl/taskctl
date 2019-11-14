package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"strings"
)

func NewRunTaskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "task [task]",
		Short: "Schedule task",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// todo: OnlyValidArgs
			var tname = args[0]
			t, ok := tasks[tname]
			if !ok {
				logrus.Fatalf("unknown task %s", tname)
			}

			var taskArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				taskArgs = args[cmd.ArgsLenAtDash():]
			}
			env := config.ConvertEnv(map[string]string{
				"ARGS": strings.Join(taskArgs, " "),
			})

			tr := runner.NewTaskRunner(contexts, env, true, quiet)
			err := tr.Run(t)
			if err != nil {
				logrus.Error(err)
			}

			close(done)
		},
	}
}
