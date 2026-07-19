package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
)

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "validate CONFIG_FILE",
		Short:   "validates config file",
		GroupID: groupInspect,
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			loader := config.NewConfigLoader(config.NewConfig())
			if _, err := loader.Load(args[0]); err != nil {
				return fmt.Errorf("invalid config; %w", err)
			}

			fmt.Println("file is valid")
			return nil
		},
	}
}
