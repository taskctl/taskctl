package output

import (
	"fmt"
	"io"

	"github.com/taskctl/taskctl/pkg/task"
)

const (
	OutputFormatRaw      = "raw"
	OutputFormatPrefixed = "prefixed"
	OutputFormatCockpit  = "cockpit"
)

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

func NewTaskOutput(t *task.Task, format string, stdout, stderr io.Writer) (*TaskOutput, error) {
	o := &TaskOutput{
		t: t,
	}

	switch format {
	case OutputFormatRaw:
		o.decorator = NewRawOutputWriter(stdout)
	case OutputFormatPrefixed:
		o.decorator = NewPrefixedOutputWriter(t, stdout)
	case OutputFormatCockpit:
		o.decorator = NewCockpitOutputWriter(t, stdout)
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
