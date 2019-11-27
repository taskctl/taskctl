package scheduler

import (
	"github.com/trntv/wilson/pkg/task"
	"sync/atomic"
)

const (
	StatusWaiting = iota
	StatusScheduled
	StatusRunning
	StatusDone
	StatusError
	StatusCanceled
)

type Stage struct {
	Name         string
	Task         task.Task
	Pipeline     string
	DependsOn    []string
	Env          map[string]string
	AllowFailure bool
	Status       int32
}

func (s *Stage) UpdateStatus(status int32) {
	atomic.StoreInt32(&s.Status, status)
}

func (s *Stage) ReadStatus() int32 {
	return atomic.LoadInt32(&s.Status)
}
