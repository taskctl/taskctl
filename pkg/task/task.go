package task

import (
	"bytes"
	"time"

	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/internal/utils"
)

type ArtifactType string

const (
	FileArtifactType   ArtifactType = "file"
	DotEnvArtifactType ArtifactType = "dotenv"
)

// Artifact holds the information about the artifact to produce
// for the specific task.
//
// NB: it is run at the end of the task so any after commands
// that mutate the output files/dotenv file will essentially
// overwrite anything set/outputted as part of the main command
type Artifact struct {
	// Name is the key under which the artifacts will be stored
	//
	// Currently this is unused
	Name string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty"`
	// Path is the glob like pattern to the
	// source of the file(s) to store as an output
	Path string `mapstructure:"path" yaml:"path" json:"path"`
	// Type is the artifact type
	// valid values are `file`|`dotenv`
	Type ArtifactType `mapstructure:"type" yaml:"type" json:"type" jsonschema:"enum=dotenv,enum=file,default=file"`
}

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
	Before       []string
	Interactive  bool
	// ResetContext is useful if multiple variations are running in the same task
	ResetContext bool
	Condition    string
	Skipped      bool

	Name        string
	Description string

	Start time.Time
	End   time.Time

	Artifacts *Artifact

	ExitCode int16
	Errored  bool
	Error    error
	Log      struct {
		Stderr *bytes.Buffer
		Stdout *bytes.Buffer
	}
}

// NewTask creates new Task instance
func NewTask(name string) *Task {
	return &Task{
		Name:      name,
		Env:       variables.NewVariables(),
		Variables: variables.NewVariables(),
		ExitCode:  -1,
		Log: struct {
			Stderr *bytes.Buffer
			Stdout *bytes.Buffer
		}{
			Stderr: &bytes.Buffer{},
			Stdout: &bytes.Buffer{},
		},
	}
}

// FromCommands creates task new Task instance with given commands
func FromCommands(name string, commands ...string) *Task {
	t := NewTask(name)
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
		return utils.LastLine(t.Log.Stderr)
	}

	return utils.LastLine(t.Log.Stdout)
}

// WithEnv sets environment variable
func (t *Task) WithEnv(key, value string) *Task {
	t.Env = t.Env.With(key, value)

	return t
}

// GetVariations returns array of maps which are task's variations
// if no variations exist one is returned to create the default job
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
