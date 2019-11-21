package cmd

import (
	"github.com/spf13/cobra"
	"log"
	"os"
)

const (
	bash_completion_func = `__wilson_parse_get()
{
    local wilson_output out
    if wilson_output=$(wilson list "$1" 2>/dev/null); then
        out=($(echo "${wilson_output}" | awk '{print $1}'))
        COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
    fi
}

__wilson_get_resource()
{
    if [[ ${#nouns[@]} -eq 0 ]]; then
        return 1
    fi
	
	echo  ${nouns[${#nouns[@]} -1]}
    __wilson_parse_get ${nouns[${#nouns[@]} -1]}
    if [[ $? -eq 0 ]]; then
        return 0
    fi
}

__custom_func() {
    case ${last_command} in
        wilson_run | wilson_run_task | wilson_watch)
            __wilson_get_resource
            return
            ;;
        *)
            ;;
    esac
}
`
)

func NewAutocompleteCommand(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [SHELL]",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"bash", "zsh"},
		Short:     "Generates completion scripts",
		Long: `To load completion run

. <(wilson completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(wilson completion)
`,
		Run: func(cmd *cobra.Command, args []string) {
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
					log.Fatal(err)
				}
			case "zsh":
				err := rootCmd.GenZshCompletion(os.Stdout)
				if err != nil {
					log.Fatal(err)
				}
			}
		},
	}
}
