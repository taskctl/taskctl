package output

import (
	"fmt"
	"io"
)

type RawOutputDecorator struct {
	w io.Writer
}

func NewRawOutputWriter(w io.Writer) *RawOutputDecorator {
	return &RawOutputDecorator{w: w}
}

func (d *RawOutputDecorator) WriteHeader() error {
	return nil
}

func (d *RawOutputDecorator) Write(b []byte) (int, error) {
	return d.w.Write(b)
}

func (d *RawOutputDecorator) WriteFooter() error {
	_, err := fmt.Fprint(d.w, "\r\n")
	return err
}
