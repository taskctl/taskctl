package cmd

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/util"
	"strings"
)

func NewRunTaskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "task [task]",
		Short: "Schedule task",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("no task specified")
			}

			_, ok := tasks[args[0]]
			if !ok {
				return errors.New("unknown task")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			t := tasks[args[0]]
			var taskArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				taskArgs = args[cmd.ArgsLenAtDash():]
			}
			env := util.ConvertEnv(map[string]string{
				"ARGS": strings.Join(taskArgs, " "),
			})

			tr := runner.NewTaskRunner(contexts, env, true, quiet)
			err := tr.Run(t)
			if err != nil {
				log.Error(err)
			}

			close(done)
		},
	}
}
