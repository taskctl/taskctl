package task

import (
	"time"

	"github.com/taskctl/taskctl/internal/util"
)

type Task struct {
	Index        uint32
	Command      []string
	Context      string
	Env          *util.Variables
	Variables    *util.Variables
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

func (t *Task) Interpolate(s string, params ...*util.Variables) (string, error) {
	data := t.Variables

	for _, variables := range params {
		data = data.Merge(variables)
	}

	return util.RenderString(s, data.Map())
}
