package task

import (
	"bytes"
	"time"

	"github.com/taskctl/taskctl/internal/variables"

	"github.com/taskctl/taskctl/internal/utils"
)

type Executable interface {
	NextCommand() interface{}
}

type Task struct {
	Index        uint32
	Command      []string
	Context      string
	Env          variables.Container
	Variables    variables.Container
	Variations   []map[string]string
	Dir          string
	Timeout      *time.Duration
	AllowFailure bool
	After        []string
	Interactive  bool

	Condition string
	Skipped   bool

	Name        string
	Description string

	Start time.Time
	End   time.Time

	ExportAs string

	ExitCode int
	Errored  bool
	Error    error
	Log      struct {
		Stderr bytes.Buffer
		Stdout bytes.Buffer
	}

	cidx int
}

func NewTask() *Task {
	return &Task{
		Env:       variables.NewVariables(nil),
		Variables: variables.NewVariables(nil),
		ExitCode:  -1,
	}
}

func FromCommand(command string) *Task {
	t := NewTask()
	t.Command = []string{command}

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
		return utils.LastLine(&t.Log.Stderr)
	}

	return utils.LastLine(&t.Log.Stdout)
}

func (t *Task) NextCommand() interface{} {
	if t.Errored {
		return nil
	}

	if t.cidx == len(t.Command) {
		return nil
	}

	c := t.Command[t.cidx]
	t.cidx++

	return c
}
