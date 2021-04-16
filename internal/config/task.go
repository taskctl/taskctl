package config

import (
	"path/filepath"

	"github.com/taskctl/taskctl/pkg/utils"
	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/task"
)

func buildTask(def *taskDefinition, lc *loaderContext) (*task.Task, error) {
	t := &task.Task{
		Name:         def.Name,
		Description:  def.Description,
		Condition:    def.Condition,
		Commands:     def.Command,
		Env:          variables.FromMap(def.Env),
		Variables:    variables.FromMap(def.Variables),
		Variations:   def.Variations,
		Dir:          def.Dir,
		Timeout:      def.Timeout,
		AllowFailure: def.AllowFailure,
		After:        def.After,
		Before:       def.Before,
		ExportAs:     def.ExportAs,
		Context:      def.Context,
		Interactive:  def.Interactive,
	}

	t.Variables.Set("Context.Name", t.Context)
	t.Variables.Set("Task.Name", t.Name)

	if def.EnvFile != "" {
		filename := def.EnvFile
		if !filepath.IsAbs(filename) && lc.Dir != "" {
			filename = filepath.Join(lc.Dir, filename)
		}

		envs, err := utils.ReadEnvFile(def.EnvFile)
		if err != nil {
			return nil, err
		}

		t.Env = variables.FromMap(envs).Merge(t.Env)
	}

	return t, nil
}
