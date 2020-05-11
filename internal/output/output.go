package output

import (
	"fmt"
	"io"
	"os"

	"github.com/taskctl/taskctl/internal/task"
)

const (
	OutputFormatRaw      = "raw"
	OutputFormatPrefixed = "prefixed"
	OutputFormatCockpit  = "cockpit"
)

var Stdout io.Writer = os.Stdout
var Stderr io.Writer = os.Stderr
var closeCh = make(chan bool)

type DecoratedOutputWriter interface {
	io.Writer
	WriteHeader() error
	WriteFooter() error
}

type TaskOutput struct {
	t         *task.Task
	decorator DecoratedOutputWriter
}

func NewTaskOutput(t *task.Task, format string) (*TaskOutput, error) {
	o := &TaskOutput{
		t: t,
	}

	switch format {
	case OutputFormatRaw:
		o.decorator = NewRawOutputWriter(Stdout)
	case OutputFormatPrefixed:
		o.decorator = NewPrefixedOutputWriter(t, Stdout)
	case OutputFormatCockpit:
		o.decorator = NewCockpitOutputWriter(t, Stdout)
	default:
		return nil, fmt.Errorf("unknown decorator \"%s\" requested", format)
	}

	return o, nil
}

func (o *TaskOutput) Stdout() io.Writer {
	return io.MultiWriter(o.decorator, &o.t.Log.Stdout)
}

func (o *TaskOutput) Stderr() io.Writer {
	return io.MultiWriter(o.decorator, &o.t.Log.Stderr)
}

func (o TaskOutput) Start() error {
	return o.decorator.WriteHeader()
}

func (o TaskOutput) Finish() error {
	return o.decorator.WriteFooter()
}

func Close() {
	close(closeCh)
}

func SetStdout(w io.Writer) {
	Stdout = w
}

func SetStderr(w io.Writer) {
	Stderr = w
}
