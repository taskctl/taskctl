package config

import (
	"os"

	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/utils"
)

func buildContext(def *ContextDefinition) (*runner.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		dir = utils.MustGetwd()
	}
	if def.Envfile != nil {
		if err := def.Envfile.Validate(); err != nil {
			return nil, err
		}
	}

	osEnvVars := variables.FromMap(utils.ConvertFromEnv(os.Environ()))
	userEnvVars := variables.FromMap(def.Env)

	// build an env order is _IMPORTANT_
	// we need to overwrite any existing env vars with the user supplied ones
	buildEnvVars := osEnvVars.Merge(userEnvVars)

	c := runner.NewExecutionContext(
		def.Executable,
		dir,
		buildEnvVars,
		def.Envfile,
		def.Up,
		def.Down,
		def.Before,
		def.After,
		runner.WithQuote(def.Quote),
	)
	c.Variables = variables.FromMap(def.Variables)

	return c, nil
}
