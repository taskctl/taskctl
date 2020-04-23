package task

import (
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/pkg/builder"
	"github.com/taskctl/taskctl/pkg/util"
)

var index uint32

type Task struct {
	Index        uint32
	Command      []string
	Context      string
	Env          []string
	Variations   []map[string]string
	Dir          string
	Timeout      *time.Duration
	AllowFailure bool
	After        []string

	Name        string
	Description string

	Start time.Time
	End   time.Time

	ExportAs string

	Errored bool
	Log     struct {
		Stderr log
		Stdout log
	}
}

func BuildTask(def *builder.TaskDefinition) *Task {
	t := &Task{
		Index:        atomic.AddUint32(&index, 1),
		Name:         def.Name,
		Description:  def.Description,
		Command:      def.Command,
		Env:          make([]string, 0),
		Variations:   def.Variations,
		Dir:          def.Dir,
		Timeout:      def.Timeout,
		AllowFailure: def.AllowFailure,
		After:        def.After,
		ExportAs:     def.ExportAs,
	}

	t.Context = def.Context
	t.Env = util.ConvertEnv(def.Env)
	if t.Context == "" {
		t.Context = "local"
	}

	return t
}

func (t *Task) Duration() time.Duration {
	if t.End.IsZero() {
		return time.Since(t.Start)
	}

	return t.End.Sub(t.Start)
}

func (t *Task) Error() string {
	if t.Log.Stderr.Len() > 0 {
		return util.LastLine(&t.Log.Stderr)
	}

	return util.LastLine(&t.Log.Stdout)
}

func (t *Task) Interpolate(s string, variables map[string]string) (string, error) {
	var vars = make(map[string]string)
	for k, v := range variables {
		vars[k] = v
	}
	vars["Name"] = t.Name

	return util.RenderString(s, vars)
}
