package output

import (
	"bufio"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/task"
)

const (
	FlavorRaw       = "raw"
	FlavorFormatted = "formatted"
	FlavorCockpit   = "cockpit"
)

var Stdout io.Writer = os.Stdout
var Stderr io.Writer = os.Stderr

type DecoratedOutputWriter interface {
	WithWriter(w io.Writer)
	Write(t *task.Task, b []byte) error
	WriteHeader(t *task.Task) error
	WriteFooter(t *task.Task) error
	Close()
}

type TaskOutput struct {
	decorator DecoratedOutputWriter
	progress  bool
	spinner   bool
	lock      sync.Mutex
}

func NewTaskOutput(flavor string, progress bool) (*TaskOutput, error) {
	var decorator DecoratedOutputWriter
	switch flavor {
	case FlavorRaw:
		decorator = NewRawOutputWriter()
	case FlavorFormatted:
		decorator = NewFormattedOutputWriter()
	case FlavorCockpit:
		decorator = NewCockpitOutputWriter()
	default:
		return nil, errors.New("unknown output flavor")
	}

	decorator.WithWriter(Stdout)
	o := &TaskOutput{
		decorator: decorator,
		progress:  progress,
	}

	return o, nil
}

type wrappedWriter struct {
	t *task.Task
	w DecoratedOutputWriter
}

func (w wrappedWriter) Write(p []byte) (n int, err error) {
	return len(p), w.w.Write(w.t, p)
}

func (o *TaskOutput) Stream(t *task.Task, flushed chan struct{}) {
	o.lock.Lock()

	o.decorator.WriteHeader(t)

	var wg sync.WaitGroup
	wg.Add(2)

	go func(dst io.Writer) {
		defer wg.Done()
		err := o.stream(dst, t.Stdout)
		if err != nil {
			logrus.Debug(err)
		}
	}(io.MultiWriter(wrappedWriter{
		t: t,
		w: o.decorator,
	}, &t.Log.Stdout))

	go func(dst io.Writer) {
		defer wg.Done()
		err := o.stream(dst, t.Stderr)
		if err != nil {
			logrus.Debug(err)
		}
	}(io.MultiWriter(wrappedWriter{
		t: t,
		w: o.decorator,
	}, &t.Log.Stderr))

	o.lock.Unlock()

	wg.Wait()

	close(flushed)
}

func (o *TaskOutput) Finish(t *task.Task) {
	o.decorator.WriteFooter(t)
}

func (o *TaskOutput) stream(dst io.Writer, src io.ReadCloser) error {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		b := scanner.Bytes()
		_, err := dst.Write(b)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (o *TaskOutput) Close() {
	o.decorator.Close()
}

func SetStdout(w io.Writer) {
	Stdout = w
}
func SetStderr(w io.Writer) {
	Stderr = w
}
