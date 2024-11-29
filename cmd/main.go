package main

import (
	"context"
	"os"
	"os/signal"
	"slices"

	taskctlcmd "github.com/Ensono/taskctl/cmd/taskctl"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func subCommands(cmd *cobra.Command) (commandNames []string) {
	for _, command := range cmd.Commands() {
		commandNames = append(commandNames, append(command.Aliases, command.Name())...)
	}
	commandNames = append(commandNames, "completion")
	return
}

// setDefaultCommandIfNonePresent This is only here for backwards compatibility
//
// If any user names a runnable task or pipeline the same as
// an existing command command will always take precedence ;)
// And will most likely fail as the argument into the command was perceived as a command name
func setDefaultCommandIfNonePresent(cmd *cobra.Command) {
	// to maintain the existing behaviour of
	// displaying a pipeline/task selector
	if len(os.Args) == 1 {
		os.Args = []string{os.Args[0], "run"}
	}

	if len(os.Args) == 2 {
		if slices.Contains([]string{"-h", "--help", "-v", "--version"}, os.Args[1]) {
			// we want the root command to display all options
			// another hack around default command
			return
		}
	}

	if len(os.Args) > 1 {
		// This will turn `taskctl [pipeline task]` => `taskctl run [pipeline task]`
		potentialCommand := os.Args[1]
		for _, command := range subCommands(cmd) {
			if command == potentialCommand {
				return
			}
		}
		os.Args = append([]string{os.Args[0], "run"}, os.Args[1:]...)
	}
}

func main() {
	taskctlRootCmd := taskctlcmd.NewTaskCtlCmd(os.Stdout, os.Stderr)

	if err := taskctlRootCmd.InitCommand(); err != nil {
		logrus.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	setDefaultCommandIfNonePresent(taskctlRootCmd.Cmd)
	if err := taskctlRootCmd.Execute(ctx); err != nil {
		logrus.Fatal(err)
	}
}
