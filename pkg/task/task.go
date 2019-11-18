package task

import (
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/util"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

const (
	STATUS_WAITING = iota
	STATUS_SCHEDULED
	STATUS_RUNNING
	STATUS_DONE
	STATUS_ERROR
	STATUS_CANCELED
)

type Task struct {
	Command []string
	Context string
	Env     []string
	Dir     string

	Name   string
	Status int32
	Start  time.Time
	End    time.Time

	Stdout io.ReadCloser
	Stderr io.ReadCloser

	stderrLastLine string

	mu sync.Mutex
}

func BuildTask(def config.TaskConfig) *Task {
	t := &Task{
		Env: make([]string, 0),
	}
	t.Command = def.Command
	t.Context = def.Context
	t.Env = util.ConvertEnv(def.Env)
	if t.Context == "" {
		t.Context = "local"
	}
	t.Dir = def.Dir

	return t
}

func (t *Task) UpdateStatus(status int32) {
	atomic.StoreInt32(&t.Status, status)
}

func (t *Task) ReadStatus() int32 {
	return atomic.LoadInt32(&t.Status)
}

func (t *Task) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawTask Task
	raw := rawTask{
		Context: "local",
	}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*t = Task(raw)
	return nil
}

func (t *Task) Duration() time.Duration {
	return t.End.Sub(t.Start)
}

func (t *Task) WiteLog(l string) {
	t.stderrLastLine = l
}

func (t *Task) ReadLog() string {
	return t.stderrLastLine
}

func (t *Task) SetStdout(stdout io.ReadCloser) {
	t.mu.Lock()
	t.Stdout = stdout
	t.mu.Unlock()
}

func (t *Task) SetStderr(stderr io.ReadCloser) {
	t.mu.Lock()
	t.Stderr = stderr
	t.mu.Unlock()
}
