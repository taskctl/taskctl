package config

import (
	"sync/atomic"

	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/util"
)

var taskIndex uint32

func buildTask(def *TaskDefinition) (*task.Task, error) {
	t := &task.Task{
		Index:        atomic.AddUint32(&taskIndex, 1),
		Name:         def.Name,
		Description:  def.Description,
		Condition:    def.Condition,
		Command:      def.Command,
		Env:          util.NewVariables(def.Env),
		Variables:    util.NewVariables(def.Variables),
		Variations:   def.Variations,
		Dir:          def.Dir,
		Timeout:      def.Timeout,
		AllowFailure: def.AllowFailure,
		After:        def.After,
		ExportAs:     def.ExportAs,
		Context:      def.Context,
	}

	if len(def.Variations) == 0 {
		// default variant
		t.Variations = make([]map[string]string, 1)
	}

	if t.Context == "" {
		t.Context = "local"
	}

	t.Variables.Set("Task.Name", t.Name)

	return t, nil
}
