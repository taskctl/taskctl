package output

import (
	"io"
	"os"
	"sync"

	"github.com/taskctl/taskctl/internal/config"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/task"
)

const (
	FlavorRaw       = config.FlavorRaw
	FlavorFormatted = config.FlavorFormatted
	FlavorCockpit   = config.FlavorCockpit
)

var Stdout io.Writer = os.Stdout
var Stderr io.Writer = os.Stderr

type DecoratedOutputWriter interface {
	Write(b []byte) (int, error)
	WriteHeader(t *task.Task) error
	WriteFooter(t *task.Task) error
	ForTask(t *task.Task) DecoratedOutputWriter
}

type TaskOutput struct {
	decorator DecoratedOutputWriter
	lock      sync.Mutex
	closeCh   chan bool
}

func NewTaskOutput(flavor string) (*TaskOutput, error) {
	o := &TaskOutput{
		closeCh: make(chan bool),
	}

	switch flavor {
	case FlavorRaw:
		o.decorator = NewRawOutputWriter(Stdout)
	case FlavorFormatted:
		o.decorator = NewFormattedOutputWriter(Stdout)
	case FlavorCockpit:
		o.decorator = NewCockpitOutputWriter(Stdout, o.closeCh)
	default:
		logrus.Error("unknown decorator requested")
	}

	return o, nil
}

func (o *TaskOutput) Stream(t *task.Task, cmdStdout, cmdStderr io.ReadCloser, flushed chan struct{}) {
	o.lock.Lock()

	var wg sync.WaitGroup
	wg.Add(2)

	decorator := o.decorator.ForTask(t)

	go func(dst io.Writer) {
		defer wg.Done()
		err := o.pipe(dst, cmdStdout)
		if err != nil {
			logrus.Debug(err)
		}
	}(io.MultiWriter(decorator, &t.Log.Stdout))

	go func(dst io.Writer) {
		defer wg.Done()
		err := o.pipe(dst, cmdStderr)
		if err != nil {
			logrus.Debug(err)
		}
	}(io.MultiWriter(decorator, &t.Log.Stderr))

	o.lock.Unlock()

	wg.Wait()
	close(flushed)
}

func (o *TaskOutput) pipe(dst io.Writer, src io.ReadCloser) error {
	var buf = make([]byte, 1)
	var err error
	for {
		_, err = src.Read(buf)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		_, err = dst.Write(buf)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *TaskOutput) Start(t *task.Task) error {
	return o.decorator.WriteHeader(t)
}

func (o *TaskOutput) Finish(t *task.Task) error {
	return o.decorator.WriteFooter(t)
}

func (o *TaskOutput) Close() {
	close(o.closeCh)
}

func SetStdout(w io.Writer) {
	Stdout = w
}
func SetStderr(w io.Writer) {
	Stderr = w
}
