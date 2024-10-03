package output

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/Ensono/taskctl/pkg/task"
	"github.com/sirupsen/logrus"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansiRegexp = regexp.MustCompile(ansi)

type prefixedOutputDecorator struct {
	t *task.Task
	w *SafeWriter
}

func NewPrefixedOutputWriter(t *task.Task, w io.Writer) *prefixedOutputDecorator {
	return &prefixedOutputDecorator{
		t: t,
		w: NewSafeWriter(w),
	}
}

// TODO: implement a chunked writer
// for when the output is too large all of a sudden
// func chunkByteSlice(items []byte, chunkSize int) (chunks [][]byte) {
// 	for chunkSize < len(items) {
// 		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
// 	}
// 	return append(chunks, items)
// }

const newLine byte = '\n'

func (d *prefixedOutputDecorator) Write(p []byte) (int, error) {
	p = ansiRegexp.ReplaceAllLiteral(p, []byte{})
	bytesWritten := 0
	br := bufio.NewReader(bytes.NewReader(p))
	for {
		line, err := br.ReadBytes(newLine)
		if err != nil && errors.Is(err, io.EOF) {
			// if the last line is empty  do not write it out
			if line != nil && len(line) == 0 {
				break
			}
			o, err := d.w.Write([]byte(fmt.Sprintf("\x1b[36m%s\x1b[0m: %s", d.t.Name, line)))
			if err != nil {
				return bytesWritten, err
			}
			bytesWritten += o
			break
		}

		// All other errors should hardstop
		if err != nil {
			return bytesWritten, err
		}
		// skip writing empty lines
		if len(line) == 0 {
			continue
		}
		o, err := d.w.Write([]byte(fmt.Sprintf("\x1b[36m%s\x1b[0m: %s", d.t.Name, line)))
		if err != nil {
			return bytesWritten, err
		}
		bytesWritten += o
	}
	return bytesWritten, nil
}

func (d *prefixedOutputDecorator) WriteHeader() error {
	logrus.Infof("Running task %s...", d.t.Name)
	return nil
}

func (d *prefixedOutputDecorator) WriteFooter() error {
	logrus.Infof("%s finished. Duration %s", d.t.Name, d.t.Duration())
	return nil
}
