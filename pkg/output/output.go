package output

import (
	"fmt"
	"io"

	"github.com/taskctl/taskctl/pkg/task"
)

// Output types
const (
	FormatRaw      = "raw"
	FormatPrefixed = "prefixed"
	FormatCockpit  = "cockpit"
)

var closed = false
var closeCh = make(chan bool)

// DecoratedOutputWriter is a decorator for task output.
// It extends io.Writer with methods to write header before output starts and footer after execution completes
type DecoratedOutputWriter interface {
	io.Writer
	WriteHeader() error
	WriteFooter() error
}

// TaskOutput connects given task with requested decorator
type TaskOutput struct {
	t         *task.Task
	decorator DecoratedOutputWriter
}

// NewTaskOutput creates new TaskOutput instance for given task.
func NewTaskOutput(t *task.Task, format string, stdout, stderr io.Writer) (*TaskOutput, error) {
	o := &TaskOutput{
		t: t,
	}

	switch format {
	case FormatRaw:
		o.decorator = newRawOutputWriter(stdout)
	case FormatPrefixed:
		o.decorator = newPrefixedOutputWriter(t, stdout)
	case FormatCockpit:
		o.decorator = newCockpitOutputWriter(t, stdout, closeCh)
	default:
		return nil, fmt.Errorf("unknown decorator \"%s\" requested", format)
	}

	return o, nil
}

// Stdout returns io.Writer that can be used for Job's STDOUT
func (o *TaskOutput) Stdout() io.Writer {
	return io.MultiWriter(o.decorator, &o.t.Log.Stdout)
}

// Stderr returns io.Writer that can be used for Job's STDERR
func (o *TaskOutput) Stderr() io.Writer {
	return io.MultiWriter(o.decorator, &o.t.Log.Stderr)
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
func Close() {
	if !closed {
		closed = true
		close(closeCh)
	}
}
