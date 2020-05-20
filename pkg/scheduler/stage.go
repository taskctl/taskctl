package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/task"
)

// Stage statuses
const (
	StatusWaiting = iota
	StatusRunning
	StatusSkipped
	StatusDone
	StatusError
	StatusCanceled
)

// Stage is a structure that describes execution stage
type Stage struct {
	Name         string
	Condition    string
	Task         *task.Task
	Pipeline     *ExecutionGraph
	DependsOn    []string
	Dir          string
	AllowFailure bool
	Status       int32
	Env          variables.Container
	Variables    variables.Container

	Start time.Time
	End   time.Time
}

// UpdateStatus updates stage's status atomically
func (s *Stage) UpdateStatus(status int32) {
	atomic.StoreInt32(&s.Status, status)
}

// ReadStatus is a helper to read stage's status atomically
func (s *Stage) ReadStatus() int32 {
	return atomic.LoadInt32(&s.Status)
}

// Duration returns stage's execution duration
func (s *Stage) Duration() time.Duration {
	return s.End.Sub(s.Start)
}
