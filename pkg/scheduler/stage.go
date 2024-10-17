package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/Ensono/taskctl/pkg/variables"

	"github.com/Ensono/taskctl/pkg/task"
)

// Stage statuses
const (
	StatusWaiting int32 = iota
	StatusRunning
	StatusSkipped
	StatusDone
	StatusError
	StatusCanceled
)

// Stage is a structure that describes execution stage
// Stage is a synonym for a Node in a the unary tree of the execution graph/tree
type Stage struct {
	Name         string
	Condition    string
	Task         *task.Task
	Pipeline     *ExecutionGraph
	DependsOn    []string
	Dir          string
	AllowFailure bool
	Status       *atomic.Int32
	Env          variables.Container
	Variables    variables.Container

	Start time.Time
	End   time.Time
}

// StageOpts is the Node options
//
// Pass in tasks/pipelines or other properties
// using the options pattern
type StageOpts func(*Stage)

func NewStage(opts ...StageOpts) *Stage {
	s := &Stage{}
	// Apply options if any
	for _, o := range opts {
		o(s)
	}
	// always overwrite and set Status here
	s.Status = &atomic.Int32{}

	return s
}

// UpdateStatus updates stage's status atomically
func (s *Stage) UpdateStatus(status int32) {
	s.Status.Store(status)
}

// ReadStatus is a helper to read stage's status atomically
func (s *Stage) ReadStatus() int32 {
	return s.Status.Load()
}

// Duration returns stage's execution duration
func (s *Stage) Duration() time.Duration {
	return s.End.Sub(s.Start)
}

type StageByStartTime []Stage

func (s StageByStartTime) Len() int {
	return len(s)
}

func (s StageByStartTime) Less(i, j int) bool {
	return s[j].Start.Nanosecond() > s[i].Start.Nanosecond()
}

func (s StageByStartTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
