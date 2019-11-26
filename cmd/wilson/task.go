package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/util"
	"strings"
)

func NewRunTaskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "task (TASK) [flags] [-- TASK_ARGS]",
		Short: "Run task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			_, err = loadConfig()
			if err != nil {
				return err
			}

			t, ok := tasks[args[0]]
			if !ok {
				return fmt.Errorf("unknown task %s", args[0])
			}
			var taskArgs []string
			if al := cmd.ArgsLenAtDash(); al > 0 {
				taskArgs = args[cmd.ArgsLenAtDash():]
			}
			env := util.ConvertEnv(map[string]string{
				"ARGS": strings.Join(taskArgs, " "),
			})

			tr := runner.NewTaskRunner(contexts, env, true, quiet)
			err = tr.Run(t)
			if err != nil {
				log.Error(err)
			}
			tr.DownContexts()

			close(done)

			return nil
		},
	}
}
