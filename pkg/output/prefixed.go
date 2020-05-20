package output

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/logrusorgru/aurora"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/task"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansiRegexp = regexp.MustCompile(ansi)

type prefixedOutputDecorator struct {
	t *task.Task
	w *bufio.Writer
}

func newPrefixedOutputWriter(t *task.Task, w io.Writer) *prefixedOutputDecorator {
	return &prefixedOutputDecorator{
		t: t,
		w: bufio.NewWriter(&lineWriter{t: t, dst: w}),
	}
}

func (d *prefixedOutputDecorator) Write(p []byte) (int, error) {
	n := len(p)
	for {
		advance, line, err := bufio.ScanLines(p, true)
		if err != nil {
			return 0, err
		}

		if advance == 0 {
			break
		}

		_, err = d.w.Write(line)
		if err != nil {
			return 0, err
		}

		err = d.w.Flush()
		if err != nil {
			return 0, err
		}

		p = p[advance:]
	}

	_, err := d.w.Write(p)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (d *prefixedOutputDecorator) WriteHeader() error {
	logrus.Infof("Running task %s...", d.t.Name)
	return nil
}

func (d *prefixedOutputDecorator) WriteFooter() error {
	err := d.w.Flush()
	if err != nil {
		logrus.Warning(err)
	}

	logrus.Infof("%s finished. Duration %s", d.t.Name, d.t.Duration())
	return nil
}

type lineWriter struct {
	t   *task.Task
	dst io.Writer
}

func (l lineWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	p = ansiRegexp.ReplaceAllLiteral(p, []byte{})
	_, err = fmt.Fprintf(l.dst, "%s: %s\r\n", aurora.Cyan(l.t.Name), p)

	return n, err
}
