package main

import (
	"context"
	"os"
	"slices"

	cmd "github.com/Ensono/taskctl/cmd/taskctl"
)

func subCommands() (commandNames []string) {
	for _, command := range cmd.TaskCtlCmd.Commands() {
		commandNames = append(commandNames, append(command.Aliases, command.Name())...)
	}
	return
}

func setDefaultCommandIfNonePresent() {
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
		for _, command := range subCommands() {
			if command == potentialCommand {
				return
			}
		}
		os.Args = append([]string{os.Args[0], "run"}, os.Args[1:]...)
	}
}

func main() {
	// This is only here for backwards compatibility
	//
	// If any user names a runnable task or pipeline the same as
	// an existing command command will always take precedence ;)
	// And will most likely fail as the argument into the command was perceived as a command name
	setDefaultCommandIfNonePresent()
	cmd.Execute(context.Background())
}
