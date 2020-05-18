package config

import (
	"os"

	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/utils"
)

type contextDefinition struct {
	Dir        string
	Up         []string
	Down       []string
	Before     []string
	After      []string
	Env        map[string]string
	Variables  map[string]string
	Executable utils.Binary
}

func buildContext(def *contextDefinition, shell *utils.Binary) (*runner.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	executable := &def.Executable

	if executable.Bin == "" && shell.Bin != "" {
		executable = shell
	}

	c := runner.NewExecutionContext(
		executable,
		dir,
		variables.FromMap(def.Env),
		def.Up,
		def.Down,
		def.Before,
		def.After,
	)

	return c, nil
}
