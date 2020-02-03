package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

func NewAutocompleteCommand(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [SHELL]",
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"bash", "zsh"},
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
			log.SetLevel(log.PanicLevel)
			log.SetOutput(ioutil.Discard)

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
				_, err := os.Stdout.Write([]byte(zshCompletion))
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

var zshCompletion = `compdef _taskctl taskctl
alias wi=taskctl
function _taskctl {
  local -a commands

  _arguments -C \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-q,--silent}'[silence output]' \
    "1: :->cmnds" \
    "*::arg:->args"

  case $state in
  cmnds)
    commands=(
      "completion:Generates completion scripts"
      "help:Help about any command"
      "list:List contexts, pipelines, tasks and watchers"
      "run:Run pipeline"
      "watch:Start watching for filesystem events"
    )
    _describe "command" commands
    ;;
  esac

  case "$words[1]" in
  completion)
    _taskctl_completion
    ;;
  help)
    _taskctl_help
    ;;
  list)
    _taskctl_list
    ;;
  run)
    _taskctl_run
    ;;
  watch)
    _taskctl_watch
    ;;
  esac
}

function _taskctl_completion {
  _arguments \
    '(-h --help)'{-h,--help}'[help for completion]' \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-s,--silent}'[silence output]' \
    '1: :("bash" "zsh")'
}

function _taskctl_help {
  _arguments \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-q --silent)'{-q,--silent}'[silence output]'
}


function _taskctl_list {
  local -a commands

  _arguments -C \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-s,--silent}'[silence output]' \
    "1: :->cmnds" \
    "*::arg:->args"

  case $state in
  cmnds)
    commands=(
      "pipelines:List pipelines"
      "tasks:List tasks"
      "watchers:List watchers"
    )
    _describe "command" commands
    ;;
  esac

  case "$words[1]" in
  pipelines)
    _taskctl_list_pipelines
    ;;
  tasks)
    _taskctl_list_tasks
    ;;
  watchers)
    _taskctl_list_watchers
    ;;
  esac
}

function _taskctl_list_pipelines {
  _arguments \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-q,--silent}'[silence output]'
}

function _taskctl_list_tasks {
  _arguments \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-s,--silent}'[silence output]'
}

function _taskctl_list_watchers {
  _arguments \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-s,--silent}'[silence output]'
}


function _taskctl_run {
  local -a commands
  pipelines=("${(@f)$(taskctl list pipelines --silent)}")
  tasks=("${(@f)$(taskctl list tasks --silent)}")

  _arguments -C \
    '--quiet[disable tasks output]' \
    '--raw-output[raw output]' \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-s --silent)'{-s,--silent}'[silence output]' \
    '(-q --quiet)'{-q,--quiet}'[disable task output]' \
    '(-s --silent)'{-s,--silent}'[silence output]' \
    "1: :->cmnds" \
    "*::arg:->args"

  case $state in
  cmnds)
    commands=("task:Run task")
    commands=($commands $pipelines $tasks)
    _describe "command" commands
    ;;
  esac

  case "$words[1]" in
  task)
    _taskctl_run_task
    ;;
  esac
}

function _taskctl_run_task {
  tasks=$(taskctl list tasks --silent | awk '{printf("\"%s\" ",$0)}')

  _arguments \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-q --quiet)'{-q,--silent}'[disable task output]' \
    '1: :('$tasks')'
}

function _taskctl_watch {
  watchers=$(taskctl list watchers --silent | awk '{printf("\"%s\" ",$0)}')

  _arguments \
    '(-c --config)'{-c,--config}'[config file to use]:filename:_files -g "yaml" -g "yml"' \
    '(-d --debug)'{-d,--debug}'[enable debug]' \
    '(-q --quiet)'{-q,--silent}'[disable task output]' \
    '1: :('$watchers')'
}
`
