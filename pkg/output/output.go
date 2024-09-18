package output

import (
	"fmt"
	"io"

	"github.com/Ensono/taskctl/pkg/task"
)

type OutputEnum string

const (
	RawOutput      OutputEnum = "raw"
	CockpitOutput  OutputEnum = "cockpit"
	PrefixedOutput OutputEnum = "prefixed"
)

// DecoratedOutputWriter is a decorator for task output.
// It extends io.Writer with methods to write header before output starts and footer after execution completes
type DecoratedOutputWriter interface {
	io.Writer // *SafeWriter
	WriteHeader() error
	WriteFooter() error
}

// TaskOutput connects given task with requested decorator
type TaskOutput struct {
	t         *task.Task
	decorator DecoratedOutputWriter
	isClosed  bool
	closeCh   chan bool
}

// NewTaskOutput creates new TaskOutput instance for given task.
func NewTaskOutput(t *task.Task, format string, stdout, stderr io.Writer) (*TaskOutput, error) {
	o := &TaskOutput{
		t:        t,
		isClosed: false,
		closeCh:  make(chan bool),
	}

	switch OutputEnum(format) {
	case RawOutput:
		o.decorator = newRawOutputWriter(stdout)
	case PrefixedOutput:
		o.decorator = NewPrefixedOutputWriter(t, stdout)
	case CockpitOutput:
		o.decorator = NewCockpitOutputWriter(t, stdout, o.closeCh)
	default:
		return nil, fmt.Errorf("unknown decorator \"%s\" requested", format)
	}

	return o, nil
}

func (t *TaskOutput) WithCloseCh(closeCh chan bool) *TaskOutput {
	t.closeCh = closeCh
	return t
}

// Stdout returns io.Writer that can be used for Job's STDOUT
func (o *TaskOutput) Stdout() io.Writer {
	return MultiWriter(o.decorator, o.t.Log.Stdout)
}

// Stderr returns io.Writer that can be used for Job's STDERR
func (o *TaskOutput) Stderr() io.Writer {
	return MultiWriter(o.decorator, o.t.Log.Stderr)
}

// Start should be called before task's output starts
func (o TaskOutput) Start() error {
	return o.decorator.WriteHeader()
}

// Finish should be called after task completes
func (o TaskOutput) Finish() error {
	return o.decorator.WriteFooter()
}

// Close releases resources and closes underlying decorators
func (t *TaskOutput) Close() {
	if !t.isClosed {
		t.isClosed = true
		close(t.closeCh)
	}
}
