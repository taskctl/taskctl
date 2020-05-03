package task

import (
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/internal/config"

	"github.com/taskctl/taskctl/internal/util"
)

var index uint32

type Task struct {
	Index        uint32
	Command      []string
	Context      string
	Env          config.Set
	Variations   []map[string]string
	Dir          string
	Timeout      *time.Duration
	AllowFailure bool
	After        []string

	Condition string
	Skipped   bool

	Name        string
	Description string

	Start time.Time
	End   time.Time

	ExportAs string

	Errored bool
	Error   error
	Log     struct {
		Stderr log
		Stdout log
	}
}

func BuildTask(def *config.TaskDefinition) *Task {
	t := &Task{
		Index:        atomic.AddUint32(&index, 1),
		Name:         def.Name,
		Description:  def.Description,
		Condition:    def.Condition,
		Command:      def.Command,
		Env:          def.Env,
		Variations:   def.Variations,
		Dir:          def.Dir,
		Timeout:      def.Timeout,
		AllowFailure: def.AllowFailure,
		After:        def.After,
		ExportAs:     def.ExportAs,
		Context:      def.Context,
	}

	if len(t.Variations) == 0 {
		// default variant
		t.Variations = make([]map[string]string, 1)
	}

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

func (t *Task) ErrorMessage() string {
	if t.Log.Stderr.Len() > 0 {
		return util.LastLine(&t.Log.Stderr)
	}

	return util.LastLine(&t.Log.Stdout)
}

func (t *Task) Interpolate(s string, params ...config.Set) (string, error) {
	data := config.NewSet(map[string]string{
		"TaskName": t.Name,
	})

	for _, variables := range params {
		data = data.Merge(variables)
	}

	return util.RenderString(s, data)
}
