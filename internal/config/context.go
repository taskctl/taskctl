package config

import (
	"github.com/taskctl/taskctl/internal/variables"
	"os"

	"github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/utils"
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

func buildContext(def *contextDefinition, shell *utils.Binary) (*context.ExecutionContext, error) {
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

	c := context.NewExecutionContext(
		executable,
		dir,
		variables.NewVariables(def.Env),
		def.Up,
		def.Down,
		def.Before,
		def.After,
	)

	return c, nil
}
