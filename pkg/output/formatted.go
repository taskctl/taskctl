package output

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/task"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

type FormattedOutputDecorator struct {
	ansiRegexp *regexp.Regexp
	w          io.Writer
}

func NewFormattedOutputWriter() *FormattedOutputDecorator {
	return &FormattedOutputDecorator{
		ansiRegexp: regexp.MustCompile(ansi),
		w:          ioutil.Discard,
	}
}

func (d *FormattedOutputDecorator) WithWriter(w io.Writer) {
	d.w = w
}

func (d *FormattedOutputDecorator) Write(t *task.Task, b []byte) error {
	bs := d.ansiRegexp.ReplaceAllLiteral(b, []byte{})
	_, err := fmt.Fprintf(d.w, "%s: %s\r\n", aurora.Cyan(t.Name), bs)

	return err
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
}
