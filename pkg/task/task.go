package task

import (
	"bytes"
	"time"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/utils"
)

// Task is a structure that describes task, its commands, environment, working directory etc.
// After task completes it provides task's execution status, exit code, stdout and stderr
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

// NewTask creates new Task instance
func NewTask() *Task {
	return &Task{
		Env:       variables.NewVariables(),
		Variables: variables.NewVariables(),
		ExitCode:  -1,
	}
}

// FromCommands creates task new Task instance with given commands
func FromCommands(commands ...string) *Task {
	t := NewTask()
	t.Commands = commands

	return t
}

// Duration returns task's execution duration
func (t *Task) Duration() time.Duration {
	if t.End.IsZero() {
		return time.Since(t.Start)
	}

	return t.End.Sub(t.Start)
}

// ErrorMessage returns message of the error occurred during task execution
func (t *Task) ErrorMessage() string {
	if !t.Errored {
		return ""
	}

	if t.Log.Stderr.Len() > 0 {
		return utils.LastLine(&t.Log.Stderr)
	}

	return utils.LastLine(&t.Log.Stdout)
}

// WithEnv sets environment variable
func (t *Task) WithEnv(key, value string) *Task {
	t.Env = t.Env.With(key, value)

	return t
}

// GetVariations returns array of maps which are task's variations
func (t *Task) GetVariations() []map[string]string {
	variations := make([]map[string]string, 1)
	if t.Variations != nil {
		variations = t.Variations
	}

	return variations
}

// Output returns task's stdout as a string
func (t *Task) Output() string {
	return t.Log.Stdout.String()
}
