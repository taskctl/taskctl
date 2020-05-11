package pipeline

import (
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/internal/utils"

	"github.com/taskctl/taskctl/internal/task"
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
	Pipeline     *ExecutionGraph
	DependsOn    []string
	Dir          string
	AllowFailure bool
	Status       int32
	Env          *utils.Variables
	Variables    *utils.Variables

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
