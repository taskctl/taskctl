package task

import (
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
	"io"
	"sync"
	"time"
)

type Task struct {
	Command []string
	Context string
	Env     []string
	Dir     string
	Timeout *time.Duration

	Name  string
	Start time.Time
	End   time.Time

	Stdout io.ReadCloser
	Stderr io.ReadCloser

	log struct {
		sync.Mutex
		data []byte
	}
}

func BuildTask(def builder.TaskDefinition) *Task {
	t := &Task{
		Name:    def.Name,
		Command: def.Command,
		Env:     make([]string, 0),
		Dir:     def.Dir,
		Timeout: def.Timeout,
	}

	t.Context = def.Context
	t.Env = util.ConvertEnv(def.Env)
	if t.Context == "" {
		t.Context = "local"
	}

	return t
}

func (t *Task) Duration() time.Duration {
	return t.End.Sub(t.Start)
}

func (t *Task) WiteLog(l []byte) {
	t.log.Lock()
	t.log.data = l
	t.log.Unlock()
}

func (t *Task) ReadLog() []byte {
	return t.log.data
}

func (t *Task) SetStdout(stdout io.ReadCloser) {
	t.Stdout = stdout
}

func (t *Task) SetStderr(stderr io.ReadCloser) {
	t.Stderr = stderr
}
