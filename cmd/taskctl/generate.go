package cmd

import (
	"fmt"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/spf13/cobra"
)

func newGenerateCmd(rootCmd *TaskCtlCmd) {
	c := &cobra.Command{
		Use:          "generate",
		Aliases:      []string{"ci", "gen-ci"},
		Short:        `generate <pipeline>`,
		Example:      `taskctl generate pipeline1`,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			// display selector if nothing is supplied
			if len(args) == 0 {
				selected, err := cmdutils.DisplayTaskSelection(conf)
				if err != nil {
					return err
				}
				args = append([]string{selected}, args[0:]...)
			}

			_, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return generateDefinition(conf, argsStringer)
		},
	}

	rootCmd.Cmd.AddCommand(c)
}

func generateDefinition(conf *config.Config, argsStringer *argsToStringsMapper) (err error) {
	graph := argsStringer.pipelineName
	if graph == nil {
		return fmt.Errorf("specified arg is not a pipeline")
	}

	graph.Generate()

	return nil
}
