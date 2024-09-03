package config

import (
	"github.com/ensono/taskctl/pkg/runner"
	"github.com/ensono/taskctl/pkg/variables"

	"github.com/ensono/taskctl/pkg/utils"
)

type contextDefinition struct {
	Dir        string
	Up         []string
	Down       []string
	Before     []string
	After      []string
	Env        map[string]string
	Envfile    utils.Envfile
	Variables  map[string]string
	Executable utils.Binary
	Quote      string
}

func buildContext(def *contextDefinition) (*runner.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		dir = utils.MustGetwd()
	}

	c := runner.NewExecutionContext(
		&def.Executable,
		dir,
		variables.FromMap(def.Env),
		&def.Envfile,
		def.Up,
		def.Down,
		def.Before,
		def.After,
		runner.WithQuote(def.Quote),
	)
	c.Variables = variables.FromMap(def.Variables)

	return c, nil
}
