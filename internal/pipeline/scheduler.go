package pipeline

import (
	"errors"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/runner"
)

type PipelineScheduler struct {
	taskRunner *runner.TaskRunner
	pause      time.Duration

	cancelled int32

	Start time.Time
	End   time.Time
}

func NewScheduler(r *runner.TaskRunner) *PipelineScheduler {
	s := &PipelineScheduler{
		pause:      50 * time.Millisecond,
		taskRunner: r,
	}

	return s
}

func (s *PipelineScheduler) Schedule(p *ExecutionGraph) error {
	s.startTimer()
	defer s.stopTimer()
	var wg = sync.WaitGroup{}

	for !s.isDone(p) {
		if atomic.LoadInt32(&s.cancelled) == 1 {
			break
		}

		for _, stage := range p.Nodes() {
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

			if !checkStatus(p, stage) {
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

				var err error
				if stage.Pipeline != nil {
					err = s.Schedule(stage.Pipeline)
				} else {
					err = s.taskRunner.Run(stage.Task, stage.Variables, stage.Env)
				}

				if err != nil {
					logrus.Error(err)
					stage.UpdateStatus(StatusError)
					if !stage.AllowFailure {
						s.Cancel()
					}

					return
				}

				stage.UpdateStatus(StatusDone)
			}(stage)
		}

		time.Sleep(s.pause)
	}

	wg.Wait()

	return p.Error()
}

func (s *PipelineScheduler) startTimer() {
	s.Start = time.Now()
}

func (s *PipelineScheduler) stopTimer() {
	s.End = time.Now()
}

func (s *PipelineScheduler) Cancel() {
	atomic.StoreInt32(&s.cancelled, 1)
	s.taskRunner.Cancel()
}

func (s *PipelineScheduler) isDone(p *ExecutionGraph) bool {
	for _, stage := range p.Nodes() {
		switch stage.ReadStatus() {
		case StatusWaiting, StatusRunning:
			return false
		}
	}

	return true
}

func (s *PipelineScheduler) Finish() {
	s.taskRunner.Finish()
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
		var e *exec.ExitError
		if errors.As(err, &e) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
