package main

import (
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewAutocompleteCommand(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [SHELL]",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"bash", "zsh", "powershell"},
		Short:     "Generates completion scripts",
		Long: `To load completion run

. <(taskctl completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(taskctl completion bash)
		
To configure your zsh shell to load completions for each session add to your zshrc

# ~/.zshrc
. <(taskctl completion zsh)
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.SetLevel(logrus.PanicLevel)
			logrus.SetOutput(ioutil.Discard)

			var shell string
			if len(args) == 0 {
				shell = "bash"
			} else {
				shell = args[0]
			}

			switch shell {
			case "bash":
				err := rootCmd.GenBashCompletion(os.Stdout)
				if err != nil {
					return err
				}
			case "zsh":
				err := rootCmd.GenZshCompletion(os.Stdout)
				if err != nil {
					return err
				}
			case "powershell":
				err := rootCmd.GenPowerShellCompletion(os.Stdout)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
