package output

import (
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/sirupsen/logrus"

	"github.com/Ensono/taskctl/pkg/task"
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
	byteSlice := bytes.Split(p, []byte{newLine})
	if len(byteSlice) == 1 {
		return d.w.Write([]byte(fmt.Sprintf("\x1b[36m%s\x1b[0m: %s\n", d.t.Name, p)))
	}
	for _, seq := range bytes.Split(p, []byte{newLine}) {
		if len(seq) == 0 {
			return bytesWritten, nil
		}
		o, err := d.w.Write([]byte(fmt.Sprintf("\x1b[36m%s\x1b[0m: %s\n", d.t.Name, seq)))
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
