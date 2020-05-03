package output

import (
	"fmt"
	"io"

	"github.com/taskctl/taskctl/internal/task"
)

type RawOutputDecorator struct {
	w io.Writer
}

func NewRawOutputWriter(w io.Writer) *RawOutputDecorator {
	return &RawOutputDecorator{w: w}
}

func (d *RawOutputDecorator) WriteHeader(t *task.Task) error {
	return nil
}

func (d *RawOutputDecorator) Write(b []byte) (int, error) {
	return d.w.Write(b)
}

func (d *RawOutputDecorator) WriteFooter(t *task.Task) error {
	_, err := fmt.Fprint(d.w, "\r\n")
	return err
}

func (d *RawOutputDecorator) ForTask(t *task.Task) DecoratedOutputWriter {
	return d
}
