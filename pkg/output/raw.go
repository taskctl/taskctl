package output

import (
	"fmt"
	"github.com/taskctl/taskctl/pkg/task"
	"io"
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

func (d *RawOutputDecorator) Close() {
}
