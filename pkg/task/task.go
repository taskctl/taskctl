package task

import (
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/sirupsen/logrus"
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
	Env          *variables.Variables
	EnvFile      *utils.Envfile
	Variables    *variables.Variables
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
	Artifacts    *Artifact

	Name        string
	Description string
	// internal fields updated by a mutex
	// only used with the single instance of the task
	mu        sync.Mutex // guards the below private fields
	start     time.Time
	end       time.Time
	skipped   bool
	exitCode  int16
	errored   bool
	errorVal  error
	Generator map[string]any
}

// NewTask creates new Task instance
func NewTask(name string) *Task {
	return &Task{
		Name:      name,
		Env:       variables.NewVariables(),
		Variables: variables.NewVariables(),
		exitCode:  -1,
		errored:   false,
		mu:        sync.Mutex{},
	}
}

func (t *Task) FromTask(task *Task) {
	if err := mergo.Merge(t, task); err != nil {
		logrus.Error("failed to dereference task")
	}
	// merge vars from preceeding higher contexts
	t.Env = t.Env.Merge(task.Env)
	t.Variables = t.Variables.Merge(task.Variables)
}

func (t *Task) WithStart(start time.Time) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.start = start
	return t
}

func (t *Task) Start() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.start
}

func (t *Task) WithEnd(end time.Time) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.end = end
	return t
}

func (t *Task) End() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.end
}

func (t *Task) WithSkipped(val bool) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.skipped = val
	return t
}

func (t *Task) Skipped() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.skipped
}

// exitCode int16
func (t *Task) WithExitCode(val int16) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.exitCode = val
	return t
}

func (t *Task) ExitCode() int16 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exitCode
}

// errored  bool
func (t *Task) WithError(val error) *Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errored = true
	t.errorVal = val
	return t
}

func (t *Task) Errored() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.errored
}

func (t *Task) Error() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.errorVal
}

// FromCommands creates task new Task instance with given commands
func FromCommands(name string, commands ...string) *Task {
	t := NewTask(name)
	t.Commands = commands
	return t
}

// Duration returns task's execution duration
func (t *Task) Duration() time.Duration {
	if t.End().IsZero() {
		return time.Since(t.Start())
	}

	return t.End().Sub(t.Start())
}

// ErrorMessage returns message of the error occurred during task execution
func (t *Task) ErrorMessage() string {
	if !t.Errored() {
		return ""
	}
	return t.Error().Error()
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
//
// This is left as a legacy method for now. will be removed in the stable 2.x versions
func (t *Task) Output() string {
	return ""
}
