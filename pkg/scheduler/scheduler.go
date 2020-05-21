package scheduler

import (
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/pkg/utils"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/runner"
)

// Scheduler executes ExecutionGraph
type Scheduler struct {
	taskRunner runner.Runner
	pause      time.Duration

	cancelled int32
}

// NewScheduler create new Scheduler instance
func NewScheduler(r runner.Runner) *Scheduler {
	s := &Scheduler{
		pause:      50 * time.Millisecond,
		taskRunner: r,
	}

	return s
}

// Schedule starts execution of the given ExecutionGraph
func (s *Scheduler) Schedule(g *ExecutionGraph) error {
	g.start = time.Now()
	defer func() { g.end = time.Now() }()

	var wg = sync.WaitGroup{}

	for !s.isDone(g) {
		if atomic.LoadInt32(&s.cancelled) == 1 {
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
			go func(stage *Stage) {
				defer func() {
					stage.End = time.Now()
					wg.Done()
				}()

				stage.Start = time.Now()

				err := s.runStage(stage)
				if err != nil {
					stage.UpdateStatus(StatusError)

					if !stage.AllowFailure {
						g.error = err
						return
					}
				}

				stage.UpdateStatus(StatusDone)
			}(stage)
		}

		time.Sleep(s.pause)
	}

	wg.Wait()

	return g.LastError()
}

// Cancel cancels executing tasks
func (s *Scheduler) Cancel() {
	atomic.StoreInt32(&s.cancelled, 1)
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
	if stage.Env != nil {
		if t.Env == nil {
			t.Env = stage.Env
		} else {
			t.Env = t.Env.Merge(stage.Env)
		}
	}

	if stage.Variables != nil {
		if t.Variables == nil {
			t.Variables = stage.Variables
		} else {
			t.Variables = t.Env.Merge(stage.Variables)
		}
	}

	return s.taskRunner.Run(stage.Task)
}

func checkStatus(p *ExecutionGraph, stage *Stage) (ready bool) {
	ready = true
	for _, dep := range p.To(stage.Name) {
		depStage, err := p.Node(dep)
		if err != nil {
			logrus.Fatal(err)
		}

		switch depStage.ReadStatus() {
		case StatusDone, StatusSkipped:
			continue
		case StatusError:
			if !depStage.AllowFailure {
				ready = false
				stage.UpdateStatus(StatusCanceled)
			}
		case StatusCanceled:
			ready = false
			stage.UpdateStatus(StatusCanceled)
		default:
			ready = false
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
