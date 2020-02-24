package task

import (
	"io"
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

	Stdout io.ReadCloser
	Stderr io.ReadCloser

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
	}

	t.Context = def.Context
	t.Env = util.ConvertEnv(def.Env)
	if t.Context == "" {
		t.Context = "local"
	}

	return t
}

func (t *Task) Duration() time.Duration {
	return t.End.Sub(t.Start)
}

func (t *Task) SetStdout(stdout io.ReadCloser) {
	t.Stdout = stdout
}

func (t *Task) SetStderr(stderr io.ReadCloser) {
	t.Stderr = stderr
}

func (t *Task) Error() string {
	if t.Log.Stderr.Len() > 0 {
		return util.LastLine(&t.Log.Stderr)
	}

	return util.LastLine(&t.Log.Stdout)
}
