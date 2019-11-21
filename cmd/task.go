package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/util"
	"strings"
)

func NewRunTaskCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "task (TASK) [flags] [-- TASK_ARGS]",
		Short:     "Run task",
		ValidArgs: util.ListNames(cfg.Tasks),
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}

			if err := cobra.OnlyValidArgs(cmd, args); err != nil {
				return err
			}

			return nil
		},
		Annotations: map[string]string{
			"cobra_annotations_zsh_completion_argument_annotation": `{
				"1": {"type": "cobra_annotations_zsh_completion_argument_word_completion", "options": ["$(wilson list tasks | awk '{printf(\"\"%\" \",$0)}')"]}
			}`,
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
			tr.DownContexts()

			close(done)
		},
	}
}
