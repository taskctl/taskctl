package output

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/logrusorgru/aurora"

	"github.com/taskctl/taskctl/pkg/task"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansiRegexp = regexp.MustCompile(ansi)

type prefixedOutputDecorator struct {
	t *task.Task
	w io.Writer
}

func newPrefixedOutputWriter(t *task.Task, w io.Writer) *prefixedOutputDecorator {
	return &prefixedOutputDecorator{
		t: t,
		w: w,
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

		_, err = d.writePrefixedLine(line)
		if err != nil {
			return 0, err
		}

		p = p[advance:]
	}

	_, err := d.writePrefixedLine(p)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (d *prefixedOutputDecorator) WriteHeader() error {
	_, err := d.writePrefixedLine([]byte(fmt.Sprintf("Running task %s...", d.t.Name)))
	return err
}

func (d *prefixedOutputDecorator) WriteFooter() error {
	_, err := d.writePrefixedLine([]byte(fmt.Sprintf("%s finished. Duration %s", d.t.Name, d.t.Duration())))
	return err
}

func (d *prefixedOutputDecorator) writePrefixedLine(p []byte) (n int, err error) {
	n = len(p)
	p = ansiRegexp.ReplaceAllLiteral(p, []byte{})
	_, err = fmt.Fprintf(d.w, "%s: %s\r\n", aurora.Cyan(d.t.Name), p)

	return n, err
}
