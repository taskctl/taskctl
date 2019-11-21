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
	StatusWaiting = iota
	StatusScheduled
	StatusRunning
	StatusDone
	StatusError
	StatusCanceled
)

type Task struct {
	Command      []string
	Context      string
	Env          []string
	Dir          string
	Timeout      *time.Duration
	AllowFailure bool

	Name   string
	Status int32
	Start  time.Time
	End    time.Time

	Stdout io.ReadCloser
	Stderr io.ReadCloser

	log struct {
		sync.Mutex
		data []byte
	}
}

func BuildTask(def config.TaskConfig) *Task {
	t := &Task{
		Command:      def.Command,
		Env:          make([]string, 0),
		Dir:          def.Dir,
		Timeout:      def.Timeout,
		AllowFailure: def.AllowFailure,
	}

	t.Context = def.Context
	t.Env = util.ConvertEnv(def.Env)
	if t.Context == "" {
		t.Context = "local"
	}

	return t
}

func (t *Task) UpdateStatus(status int32) {
	atomic.StoreInt32(&t.Status, status)
}

func (t *Task) ReadStatus() int32 {
	return atomic.LoadInt32(&t.Status)
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
