package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/variables"

	"github.com/taskctl/taskctl/task"
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
	status       atomic.Int32
	Env          variables.Container
	Variables    variables.Container

	Start time.Time
	End   time.Time
}

// updateStatus updates stage's status atomically
func (s *Stage) updateStatus(status int32) {
	s.status.Store(status)
}

// ReadStatus is a helper to read stage's status atomically
func (s *Stage) ReadStatus() int32 {
	return s.status.Load()
}

// Duration returns stage's execution duration
func (s *Stage) Duration() time.Duration {
	return s.End.Sub(s.Start)
}
