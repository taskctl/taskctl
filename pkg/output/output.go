package output

import (
	"fmt"
	"io"
	"sync"

	"github.com/Ensono/taskctl/pkg/task"
)

type OutputEnum string

const (
	RawOutput      OutputEnum = "raw"
	CockpitOutput  OutputEnum = "cockpit"
	PrefixedOutput OutputEnum = "prefixed"
)

type SafeWriter struct {
	writerImpl   io.Writer
	bytesWritten []byte
	mu           *sync.Mutex
}

// NewSafeWriter initiates a new concurrency safe writer
func NewSafeWriter(writerImpl io.Writer) *SafeWriter {
	return &SafeWriter{writerImpl: writerImpl, bytesWritten: []byte{}, mu: &sync.Mutex{}}
}

func (tw *SafeWriter) Write(p []byte) (n int, err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.bytesWritten = append(tw.bytesWritten, p...)
	return tw.writerImpl.Write(p)
	// return len(p), nil
}

func (tw *SafeWriter) String() string {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return string(tw.bytesWritten)
}

func (tw *SafeWriter) Len() int {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return len(tw.bytesWritten)
}

// var closed = false
// var closeCh = make(chan bool)

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
		o.decorator = newRawOutputWriter(NewSafeWriter(stdout))
	case PrefixedOutput:
		o.decorator = NewPrefixedOutputWriter(t, NewSafeWriter(stdout))
	case CockpitOutput:
		o.decorator = NewCockpitOutputWriter(t, NewSafeWriter(stdout), o.closeCh)
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
	return io.MultiWriter(o.decorator, o.t.Log.Stdout)
}

// Stderr returns io.Writer that can be used for Job's STDERR
func (o *TaskOutput) Stderr() io.Writer {
	return io.MultiWriter(o.decorator, o.t.Log.Stderr)
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
