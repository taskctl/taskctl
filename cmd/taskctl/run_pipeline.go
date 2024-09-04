package cmd

import "github.com/spf13/cobra"

var (
	runPipelineCmd = &cobra.Command{
		Use:   "pipeline",
		Short: `runs pipeline <task>`,
		Long:  `taskctl pipeline task1`,
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return buildTaskRunner(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(pipelineName, taskRunner, conf.Summary)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			postRunReset()
			return nil
		},
	}
)

func init() {
	runCmd.AddCommand(runPipelineCmd)
}
