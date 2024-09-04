package cmd

import "github.com/spf13/cobra"

var (
	runTaskCmd = &cobra.Command{
		Use:     "task",
		Aliases: []string{},
		Short:   `runs task <task>`,
		Long:    `taskctl run task1`,
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return buildTaskRunner(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTask(taskName, taskRunner)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			postRunReset()
			return nil
		},
	}
)

func init() {
	runCmd.AddCommand(runTaskCmd)
}
