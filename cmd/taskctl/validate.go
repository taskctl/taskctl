package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateCmd(rootCmd *TaskCtlCmd) {
	c := &cobra.Command{
		Use:   "validate",
		Short: `validates config file`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			fmt.Fprintln(rootCmd.ChannelOut, "file is valid")
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return nil // postRunReset()
		},
	}
	rootCmd.Cmd.AddCommand(c)
}
