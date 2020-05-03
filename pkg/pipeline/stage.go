package pipeline

import (
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/pkg/config"

	"github.com/taskctl/taskctl/pkg/task"
)

const (
	StatusWaiting = iota
	StatusRunning
	StatusSkipped
	StatusDone
	StatusError
	StatusCanceled
)

type Stage struct {
	Name         string
	Condition    string
	Task         *task.Task
	Pipeline     *Pipeline
	DependsOn    []string
	Env          map[string]string
	Dir          string
	AllowFailure bool
	Status       int32
	Variables    config.Set

	Start time.Time
	End   time.Time
}

func (s *Stage) UpdateStatus(status int32) {
	atomic.StoreInt32(&s.Status, status)
}

func (s *Stage) ReadStatus() int32 {
	return atomic.LoadInt32(&s.Status)
}

func (s *Stage) Duration() time.Duration {
	return s.End.Sub(s.Start)
}

func (s *Stage) SetEnvVariable(name, value string) {
	if s.Env == nil {
		s.Env = make(map[string]string)
	}
	s.Env[name] = value
}
