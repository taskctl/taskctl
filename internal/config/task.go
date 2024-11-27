package config

import (
	"dario.cat/mergo"
	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/task"
)

func buildTask(def *TaskDefinition, lc *loaderContext) (*task.Task, error) {

	t := task.NewTask(def.Name)

	t.Description = def.Description
	t.Condition = def.Condition
	t.Commands = def.Command
	t.Variations = def.Variations
	t.Timeout = def.Timeout
	t.AllowFailure = def.AllowFailure
	t.After = def.After
	t.Before = def.Before
	t.Artifacts = def.Artifacts
	t.Context = def.Context
	t.Interactive = def.Interactive
	t.ResetContext = def.ResetContext

	t.Env = variables.FromMap(def.Env).Merge(t.Env)
	ef := utils.NewEnvFile()

	if def.Envfile != nil {
		_ = mergo.Merge(ef, def.Envfile)
	}
	t.EnvFile = ef

	t.Variables = variables.FromMap(def.Variables).Merge(t.Variables)

	t.Dir = def.Dir
	if def.Dir == "" {
		t.Dir = lc.Dir
	}

	t.Variables.Set("Context.Name", t.Context)
	t.Variables.Set("Task.Name", t.Name)

	// Generator CI YAML
	t.Generator = def.Generator

	return t, nil
}
