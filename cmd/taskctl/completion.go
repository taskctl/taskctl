package main

import (
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func newCompletionCommand() *cli.Command {
	return &cli.Command{
		Name:      "completion",
		Usage:     "generates completion scripts",
		UsageText: helpText,
		Action: func(c *cli.Context) error {
			logrus.SetLevel(logrus.PanicLevel)
			logrus.SetOutput(ioutil.Discard)

			var shell string
			if !c.Args().Present() {
				shell = "bash"
			} else {
				shell = c.Args().First()
			}

			fmt.Println("PROG=taskctl")
			switch shell {
			case "bash":
				fmt.Println(bashSource)
			case "zsh":
				fmt.Println("_CLI_ZSH_AUTOCOMPLETE_HACK=1")
				fmt.Println(zshSource)
			default:
				return fmt.Errorf("unsupported shell type")
			}

			return nil
		},
	}
}

var helpText = `
To load completion run

. <(taskctl completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(taskctl completion bash)
		
To configure your zsh shell to load completions for each session add to your zshrc

# ~/.zshrc
. <(taskctl completion zsh)
`

var bashSource = `#! /bin/bash

: ${PROG:=$(basename ${BASH_SOURCE})}

_cli_bash_autocomplete() {
  if [[ "${COMP_WORDS[0]}" != "source" ]]; then
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == "-"* ]]; then
      opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} ${cur} --generate-bash-completion )
    else
      opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
    fi
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi
}

complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete $PROG
unset PROG
`
var zshSource = `#compdef $PROG

_cli_zsh_autocomplete() {

  local -a opts
  local cur
  cur=${words[-1]}
  if [[ "$cur" == "-"* ]]; then
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion)}")
  else
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} --generate-bash-completion)}")
  fi

  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  fi

  return
}

compdef _cli_zsh_autocomplete $PROG
`
