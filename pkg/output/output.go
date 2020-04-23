package output

import (
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
	Write(b []byte) (int, error)
	WriteHeader(t *task.Task) error
	WriteFooter(t *task.Task) error
}

type TaskOutput struct {
	flavor   string
	progress bool
	spinner  bool
	lock     sync.Mutex
	closeCh  chan bool
}

func NewTaskOutput(flavor string, progress bool) (*TaskOutput, error) {
	o := &TaskOutput{
		flavor:   flavor,
		progress: progress,
		closeCh:  make(chan bool),
	}

	return o, nil
}

func (o *TaskOutput) Stream(t *task.Task, cmdStdout, cmdStderr io.ReadCloser, flushed chan struct{}) {
	o.lock.Lock()

	var decorator DecoratedOutputWriter
	switch o.flavor {
	case FlavorRaw:
		decorator = NewRawOutputWriter(Stdout)
	case FlavorFormatted:
		decorator = NewFormattedOutputWriter(Stdout, t)
	case FlavorCockpit:
		decorator = NewCockpitOutputWriter(Stdout, o.closeCh)
	default:
		logrus.Error("unknown decorator requested")
	}

	decorator.WriteHeader(t)

	var wg sync.WaitGroup
	wg.Add(2)

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
	decorator.WriteFooter(t)
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

func (o *TaskOutput) Close() {
	close(o.closeCh)
}

func SetStdout(w io.Writer) {
	Stdout = w
}
func SetStderr(w io.Writer) {
	Stderr = w
}
