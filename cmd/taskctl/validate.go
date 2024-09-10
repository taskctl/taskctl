package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: `validates config file`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgFilePath = args[0]
			if err := initConfig(); err != nil {
				return err
			}
			fmt.Fprintln(ChannelOut, "file is valid")
			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
)

func init() {
	TaskCtlCmd.AddCommand(validateCmd)
}
