package output

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/task"
)

const ansiPattern = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var ansiRegexp = regexp.MustCompile(ansiPattern)

type prefixedOutputDecorator struct {
	t      *task.Task
	w      io.Writer
	prefix string
}

func newPrefixedOutputWriter(t *task.Task, w io.Writer) *prefixedOutputDecorator {
	return &prefixedOutputDecorator{
		t:      t,
		w:      tui.NewWriter(w),
		prefix: tui.StylePrefix.Render(t.Name),
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

		err = d.writePrefixedLine(line)
		if err != nil {
			return 0, err
		}

		p = p[advance:]
	}

	err := d.writePrefixedLine(p)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (d *prefixedOutputDecorator) WriteHeader() error {
	err := d.writePrefixedLine(fmt.Appendf(nil, "Running task %s...", d.t.Name))
	return err
}

func (d *prefixedOutputDecorator) WriteFooter() error {
	err := d.writePrefixedLine(fmt.Appendf(nil, "%s finished. Duration %s", d.t.Name, d.t.Duration()))
	return err
}

func (d *prefixedOutputDecorator) writePrefixedLine(p []byte) error {
	p = ansiRegexp.ReplaceAllLiteral(p, []byte{})
	_, err := fmt.Fprintf(d.w, "%s: %s\r\n", d.prefix, p)

	return err
}
