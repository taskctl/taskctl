package output

import (
	"fmt"
	"github.com/taskctl/taskctl/pkg/task"
	"io"
	"io/ioutil"
)

type RawOutputDecorator struct {
	w io.Writer
}

func NewRawOutputWriter() *RawOutputDecorator {
	return &RawOutputDecorator{w: ioutil.Discard}
}

func (d *RawOutputDecorator) WithWriter(w io.Writer) {
	d.w = w
}

func (d *RawOutputDecorator) WriteHeader(t *task.Task) error {
	return nil
}

func (d *RawOutputDecorator) Write(t *task.Task, b []byte) error {
	_, err := fmt.Fprintln(d.w, string(b))

	return err
}

func (d *RawOutputDecorator) WriteFooter(t *task.Task) error {
	_, err := fmt.Fprint(d.w, "\r\n")
	return err
}

func (d *RawOutputDecorator) Close() {
}
