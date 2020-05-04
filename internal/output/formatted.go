package output

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/logrusorgru/aurora"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/task"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansiRegexp = regexp.MustCompile(ansi)

type FormattedOutputDecorator struct {
	w   io.Writer
	buf *bufio.Writer
}

func NewPrefixedOutputWriter(w io.Writer) *FormattedOutputDecorator {
	return &FormattedOutputDecorator{
		w:   w,
		buf: bufio.NewWriter(w),
	}
}

func (d *FormattedOutputDecorator) Write(p []byte) (int, error) {
	if d.buf.Available() == 0 || bytes.IndexByte(p, '\n') >= 0 {
		err := d.buf.Flush()
		if err != nil {
			return 0, err
		}
	}

	return d.buf.Write(p)
}

func (d *FormattedOutputDecorator) WriteHeader(t *task.Task) error {
	logrus.Infof("Running task %s...", t.Name)
	return nil
}

func (d *FormattedOutputDecorator) WriteFooter(t *task.Task) error {
	logrus.Infof("%s finished. Duration %s", t.Name, t.Duration())
	return nil
}

func (d *FormattedOutputDecorator) Close() {
	err := d.buf.Flush()
	if err != nil {
		logrus.Warning(err)
	}
}

func (d FormattedOutputDecorator) ForTask(t *task.Task) DecoratedOutputWriter {
	d.buf = bufio.NewWriter(&lineWriter{t: t, out: d.w})
	return &d
}

type lineWriter struct {
	t   *task.Task
	out io.Writer
}

func (l lineWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	p = ansiRegexp.ReplaceAllLiteral(p, []byte{})
	p = bytes.Trim(p, "\r\n")
	_, err = fmt.Fprintf(l.out, "%s: %s\r\n", aurora.Cyan(l.t.Name), p)

	return n, err
}
