package runner

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/task"
	"io"
	"io/ioutil"
	"os"
)

type taskOutput struct {
	raw   bool
	quiet bool

	stdout io.Writer
	stderr io.Writer
}

func NewTaskOutput(raw bool, quiet bool) *taskOutput {
	o := &taskOutput{
		raw:   raw,
		quiet: quiet,
	}

	if quiet {
		o.stdout = ioutil.Discard
		o.stderr = ioutil.Discard
	} else {
		o.stdout = os.Stdout
		o.stderr = os.Stderr
	}

	return o
}

func (o *taskOutput) Scan(t *task.Task, done chan struct{}, flushed chan struct{}) {
	o.streamOutput(t, done)

	d, err := ioutil.ReadAll(t.Stdout)
	if len(d) > 0 && err != nil {
		_, err = fmt.Fprintf(o.stdout, "%s: %s\r\n", t.Name, d)
		if err != nil {
			logrus.Debug(err)
		}
	}

	d, err = ioutil.ReadAll(t.Stderr)
	if len(d) > 0 && err != nil {
		_, err = fmt.Fprintf(o.stderr, "%s: %s\r\n", aurora.Red(t.Name), d)
		if err != nil {
			logrus.Debug(err)
		}
	}

	close(flushed)
}

func (o *taskOutput) streamOutput(t *task.Task, done chan struct{}) {
	for {
		if t.ReadStatus() != task.STATUS_RUNNING {
			return
		}
		select {
		case <-done:
			return
		default:
			if o.raw {
				o.streamRawOutput(t)
			} else {
				err := o.streamDecoratedStdoutOutput(t)
				if err != nil {
					return
				}
				err = o.streamDecoratedStderrOutput(t)
				if err != nil {
					return
				}
			}
		}
	}
}

func (o *taskOutput) streamRawOutput(t *task.Task) {
	logw, logr, _ := os.Pipe()
	stderr := io.MultiWriter(o.stderr, logw)

	_, err := io.Copy(stderr, t.Stdout)
	if err != nil {
		logrus.Debug(err)
	}

	scanner := bufio.NewScanner(logr)
	for scanner.Scan() {
		t.WiteLog(scanner.Text())
	}
}

func (o *taskOutput) streamDecoratedStdoutOutput(t *task.Task) error {
	scanner := bufio.NewScanner(t.Stdout)
	for scanner.Scan() {
		_, err := fmt.Fprintf(o.stdout, "%s: %s\r\n", t.Name, scanner.Text())
		if err != nil {
			logrus.Debug(err)
		}
	}

	if err := scanner.Err(); err != nil {
		if !errors.Is(err, os.ErrClosed) {
			return err
		}
	}

	return nil
}

func (o *taskOutput) streamDecoratedStderrOutput(t *task.Task) error {
	scanner := bufio.NewScanner(t.Stderr)
	for scanner.Scan() {
		line := scanner.Text()
		t.WiteLog(line)
		_, err := fmt.Fprintf(o.stderr, "%s: %s\r\n", aurora.Red(t.Name), line)
		if err != nil {
			logrus.Debug(err)
		}
	}

	if err := scanner.Err(); err != nil {
		if !errors.Is(err, os.ErrClosed) {
			return err
		}
	}

	return nil
}
