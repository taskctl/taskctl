// Package scheduler ensures all tasks in a pipeline and child pipelines
// are loaded and executed in the order they need to be, parallelizing where possible.
package scheduler

import (
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/sirupsen/logrus"
)

// Scheduler executes ExecutionGraph
type Scheduler struct {
	taskRunner runner.Runner
	pause      time.Duration
	cancelled  *atomic.Int32
}

// NewScheduler create new Scheduler instance
func NewScheduler(r runner.Runner) *Scheduler {
	s := &Scheduler{
		pause:      50 * time.Millisecond,
		taskRunner: r,
		cancelled:  &atomic.Int32{},
	}

	return s
}

func (s *Scheduler) Cancelled() int32 {
	return s.cancelled.Load()
}

// Schedule starts execution of the given ExecutionGraph
func (s *Scheduler) Schedule(g *ExecutionGraph) error {
	g.mu.Lock()
	g.start = time.Now()
	g.mu.Unlock()
	defer func() {
		g.mu.Lock()
		g.end = time.Now()
		g.mu.Unlock()
	}()

	wg := sync.WaitGroup{}

	for !s.isDone(g) {
		if s.Cancelled() == 1 {
			break
		}

		for _, stage := range g.Nodes() {
			status := stage.ReadStatus()
			if status != StatusWaiting {
				continue
			}

			if stage.Condition != "" {
				meets, err := checkStageCondition(stage.Condition)
				if err != nil {
					logrus.Error(err)
					stage.UpdateStatus(StatusError)
					s.Cancel()
					continue
				}

				if !meets {
					stage.UpdateStatus(StatusSkipped)
					continue
				}
			}

			if !checkStatus(g, stage) {
				continue
			}

			wg.Add(1)
			stage.UpdateStatus(StatusRunning)
			go func(stage *Stage, wg *sync.WaitGroup) {
				defer func() {
					stage.WithEnd(time.Now())
					wg.Done()
				}()

				stage.WithStart(time.Now())

				err := s.runStage(stage)
				if err != nil {
					stage.UpdateStatus(StatusError)

					if !stage.AllowFailure {
						g.error = err
						return
					}
				}

				stage.UpdateStatus(StatusDone)
			}(stage, &wg)
		}

		time.Sleep(s.pause)
	}

	wg.Wait()

	return g.LastError()
}

// Cancel cancels executing tasks
// atomically loads the cancelled code
func (s *Scheduler) Cancel() {
	s.cancelled.Store(1)
	s.taskRunner.Cancel()
}

// Finish finishes scheduler's TaskRunner
func (s *Scheduler) Finish() {
	s.taskRunner.Finish()
}

func (s *Scheduler) isDone(p *ExecutionGraph) bool {
	for _, stage := range p.Nodes() {
		switch stage.ReadStatus() {
		case StatusWaiting, StatusRunning:
			return false
		}
	}

	return true
}

func (s *Scheduler) runStage(stage *Stage) error {
	if stage.Pipeline != nil {
		return s.Schedule(stage.Pipeline)
	}

	t := stage.Task
	// Precedence setter of env and vars
	// Context > Pipeline > Task
	t.Env = t.Env.Merge(stage.Env())
	t.Variables = t.Variables.Merge(stage.Variables())

	return s.taskRunner.Run(t)
}

// checkStatus checks the parents of the stage
// when they are all completed or skipped
// the task is marked as ready for execution
func checkStatus(p *ExecutionGraph, stage *Stage) bool {
	ready := false
	for _, parentStage := range p.Parents(stage.Name) {
		switch parentStage.ReadStatus() {
		case StatusDone, StatusSkipped:
			// status remains as ready
			ready = true
			continue
		case StatusError:
			if !parentStage.AllowFailure {
				stage.UpdateStatus(StatusCanceled)
			}
		case StatusCanceled:
			stage.UpdateStatus(StatusCanceled)
		}
	}
	return ready
}

func checkStageCondition(condition string) (bool, error) {
	cmd := exec.Command(condition)
	err := cmd.Run()
	if err != nil {
		if utils.IsExitError(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
