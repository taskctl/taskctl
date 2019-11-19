package runner

import (
	"bufio"
	"fmt"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
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

type logWriter struct {
	t *task.Task
}

func (l logWriter) Write(p []byte) (n int, err error) {
	l.t.WiteLog(p)

	return len(p), nil
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
			log.Debug(err)
		}
	}

	d, err = ioutil.ReadAll(t.Stderr)
	if len(d) > 0 && err != nil {
		_, err = fmt.Fprintf(o.stderr, "%s: %s\r\n", aurora.Red(t.Name), d)
		if err != nil {
			log.Debug(err)
		}
	}

	close(flushed)
}

func (o *taskOutput) streamOutput(t *task.Task, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			if o.raw {
				err := o.streamRawOutput(t)
				if err != nil {
					return
				}
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

func (o *taskOutput) streamRawOutput(t *task.Task) error {
	lw := &logWriter{t: t}
	err := o.stream(o.stdout, t.Stdout, lw)
	if err != nil {
		return err
	}

	err = o.stream(o.stderr, t.Stderr, lw)
	if err != nil {
		return err
	}

	return nil
}

func (o *taskOutput) stream(dst io.Writer, src io.ReadCloser, log io.Writer) error {
	logw, logr, _ := os.Pipe()
	w := io.MultiWriter(dst, logw)

	_, err := io.Copy(w, src)
	if err == os.ErrClosed {
		return err
	}

	scanner := bufio.NewScanner(logr)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) > 0 {
			_, err = log.Write(b)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *taskOutput) streamDecoratedStdoutOutput(t *task.Task) error {
	scanner := bufio.NewScanner(t.Stdout)
	for scanner.Scan() {
		_, err := fmt.Fprintf(o.stdout, "%s: %s\r\n", t.Name, scanner.Text())
		if err != nil {
			log.Debug(err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (o *taskOutput) streamDecoratedStderrOutput(t *task.Task) error {
	scanner := bufio.NewScanner(t.Stderr)
	for scanner.Scan() {
		line := scanner.Text()
		t.WiteLog([]byte(line))
		_, err := fmt.Fprintf(o.stderr, "%s: %s\r\n", aurora.Red(t.Name), line)
		if err != nil {
			log.Debug(err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
