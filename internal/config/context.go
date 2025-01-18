package config

import (
	"path/filepath"

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
	EnvFile    string `mapstructure:"env_file"`
	Variables  map[string]string
	Executable utils.Binary
	Quote      string
}

func buildContext(def *contextDefinition) (*runner.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		dir = utils.MustGetwd()
	}

	envs := variables.FromMap(def.Env)
	if def.EnvFile != "" {
		filename := def.EnvFile
		if !filepath.IsAbs(filename) && dir != "" {
			filename = filepath.Join(dir, filename)
		}

		fileEnvs, err := utils.ReadEnvFile(filename)
		if err != nil {
			return nil, err
		}

		envs = variables.FromMap(fileEnvs).Merge(envs)
	}

	c := runner.NewExecutionContext(
		&def.Executable,
		dir,
		envs,
		def.Up,
		def.Down,
		def.Before,
		def.After,
		runner.WithQuote(def.Quote),
	)
	c.Variables = variables.FromMap(def.Variables)

	return c, nil
}
