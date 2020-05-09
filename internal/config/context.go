package config

import (
	"os"

	"github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/util"
)

type contextDefinition struct {
	Dir       string
	Up        []string
	Down      []string
	Before    []string
	After     []string
	Env       map[string]string
	Variables map[string]string
	util.Executable
}

func buildContext(def *contextDefinition, shell util.Executable) (*context.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	executable := util.Executable{
		Bin:  def.Executable.Bin,
		Args: def.Executable.Args,
	}

	if executable.Bin == "" {
		if shell.Bin != "" {
			executable = shell
		} else {
			executable = defaultShell()
		}
	}

	c := context.NewExecutionContext(
		executable,
		dir,
		append(os.Environ(), util.ConvertEnv(def.Env)...),
		def.Up,
		def.Down,
		def.Before,
		def.After,
	)

	return c, nil
}

func defaultShell() util.Executable {
	return util.Executable{
		Bin:  "/bin/sh",
		Args: []string{"-c"},
	}
}
