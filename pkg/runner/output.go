package runner

import (
	"bufio"
	"fmt"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/pkg/task"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sync"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

type taskOutput struct {
	raw   bool
	quiet bool

	stdout io.Writer
	stderr io.Writer

	ansiRegexp *regexp.Regexp
}

type logWriter struct {
	t *task.Task
}

type linearWriter struct {
	dst io.Writer
}

func (l logWriter) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		l.t.WriteLog(p)
	}

	return len(p), nil
}

func (l linearWriter) Write(p []byte) (n int, err error) {
	_, err = fmt.Fprintln(l.dst, string(p))
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func NewTaskOutput(raw bool, quiet bool) *taskOutput {
	o := &taskOutput{
		raw:        raw,
		quiet:      quiet,
		ansiRegexp: regexp.MustCompile(ansi),
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

func (o *taskOutput) Scan(t *task.Task, flushed chan struct{}) {
	lw := &logWriter{t: t}

	var wg sync.WaitGroup
	wg.Add(2)
	if o.raw {
		go func() {
			defer wg.Done()
			err := o.streamRawOutput(&linearWriter{dst: o.stdout}, t.Stdout, lw)
			if err != nil {
				log.Debug(err)
			}
		}()

		go func() {
			defer wg.Done()
			err := o.streamRawOutput(&linearWriter{dst: o.stdout}, t.Stderr, lw)
			if err != nil {
				log.Debug(err)
			}
		}()
	} else {
		go func() {
			defer wg.Done()
			err := o.streamDecoratedOutput(t, o.stdout, t.Stdout, lw)
			if err != nil {
				log.Debug(err)
			}
		}()

		go func() {
			defer wg.Done()
			err := o.streamDecoratedOutput(t, o.stderr, t.Stderr, lw)
			if err != nil {
				log.Debug(err)
			}
		}()
	}

	wg.Wait()
	close(flushed)
}

func (o *taskOutput) streamRawOutput(dst io.Writer, src io.ReadCloser, lw io.Writer) error {
	w := io.MultiWriter(dst, lw)

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		b := scanner.Bytes()
		_, err := w.Write(b)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (o *taskOutput) streamDecoratedOutput(t *task.Task, dst io.Writer, src io.ReadCloser, lw io.Writer) error {
	w := io.MultiWriter(dst, lw)

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		b := scanner.Bytes()
		bs := o.ansiRegexp.ReplaceAllLiteral(b, []byte{})
		_, err := fmt.Fprintf(w, "%s: %s\r\n", aurora.Cyan(t.Name), bs)
		if err != nil {
			log.Debug(err)
		}

		_, err = lw.Write(b)
		if err != nil {
			log.Debug(err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
