package output

import (
	"bufio"
	"fmt"
	"io"

	"github.com/Ensono/taskctl/pkg/task"
	"github.com/sirupsen/logrus"
)

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

func (d *prefixedOutputDecorator) Write(p []byte) (int, error) {
	n := len(p)
	for {
		// use ScanLines for an easier newlint and empty output management
		advance, line, err := bufio.ScanLines(p, true)
		// All errors should hardstop
		if err != nil {
			return 0, err
		}
		if advance == 0 {
			break
		}
		// do not write empty lines
		if len(line) == 0 {
			break
		}
		if _, err := d.w.Write([]byte(fmt.Sprintf("\x1b[36m%s\x1b[0m: %s\r\n", d.t.Name, line))); err != nil {
			return 0, err
		}
		p = p[advance:]
	}
	return n, nil
}

func (d *prefixedOutputDecorator) WriteHeader() error {
	logrus.Infof("Running task %s...", d.t.Name)
	return nil
}

func (d *prefixedOutputDecorator) WriteFooter() error {
	logrus.Infof("%s finished. Duration %s", d.t.Name, d.t.Duration())
	return nil
}
