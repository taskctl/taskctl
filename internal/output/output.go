package output

import (
	"fmt"
	"io"

	"github.com/taskctl/taskctl/task"
)

// Output types
const (
	FormatRaw      = "raw"
	FormatPrefixed = "prefixed"
	FormatDefault  = "default"
	FormatJSON     = "json"
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

// streamAwareWriter is implemented by decorators that need to attribute
// writes to a specific named stream (e.g. "stdout" vs "stderr") rather than
// treating all output the same way. TaskOutput.Stdout()/Stderr() type-assert
// the decorator against this interface and, when implemented, route writes
// through the returned stream-specific writer instead of the decorator itself.
type streamAwareWriter interface {
	StreamWriter(stream string) io.Writer
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
	case FormatDefault:
		o.decorator = newDashboardOutputWriter(t, stdout, closeCh)
	case FormatJSON:
		o.decorator = newJSONOutputWriter(t, stdout)
	default:
		return nil, fmt.Errorf("unknown decorator \"%s\" requested", format)
	}

	return o, nil
}

// Stdout returns io.Writer that can be used for Job's STDOUT
func (o *TaskOutput) Stdout() io.Writer {
	if sa, ok := o.decorator.(streamAwareWriter); ok {
		return io.MultiWriter(sa.StreamWriter("stdout"), &o.t.Log.Stdout)
	}
	return io.MultiWriter(o.decorator, &o.t.Log.Stdout)
}

// Stderr returns io.Writer that can be used for Job's STDERR
func (o *TaskOutput) Stderr() io.Writer {
	if sa, ok := o.decorator.(streamAwareWriter); ok {
		return io.MultiWriter(sa.StreamWriter("stderr"), &o.t.Log.Stderr)
	}
	return io.MultiWriter(o.decorator, &o.t.Log.Stderr)
}

// Start should be called before task's output starts
func (o *TaskOutput) Start() error {
	return o.decorator.WriteHeader()
}

// Finish should be called after task completes
func (o *TaskOutput) Finish() error {
	return o.decorator.WriteFooter()
}

// Close releases resources and closes underlying decorators. For the default
// (dashboard) output it blocks until the dashboard program has fully shut down
// (final lines flushed, terminal restored).
func Close() {
	if closed {
		return
	}
	closed = true

	baseMu.Lock()
	b := base
	baseMu.Unlock()

	if b == nil {
		return
	}
	close(b.closeCh)
	b.wait()
}
