package task

import (
	"bytes"
	"time"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/utils"
)

type Executable interface {
	NextCommand() interface{}
}

type Task struct {
	Commands     []string // Commands to run
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

	ExitCode int16
	Errored  bool
	Error    error
	Log      struct {
		Stderr bytes.Buffer
		Stdout bytes.Buffer
	}
}

func NewTask() *Task {
	return &Task{
		Env:       variables.NewVariables(),
		Variables: variables.NewVariables(),
		ExitCode:  -1,
	}
}

func FromCommands(commands ...string) *Task {
	t := NewTask()
	t.Commands = commands

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

func (t *Task) WithEnv(key, value string) *Task {
	t.Env = t.Env.With(key, value)

	return t
}

func (t *Task) GetVariations() []map[string]string {
	variations := make([]map[string]string, 1)
	if t.Variations != nil {
		variations = t.Variations
	}

	return variations
}

func (t *Task) Output() string {
	return t.Log.Stdout.String()
}
