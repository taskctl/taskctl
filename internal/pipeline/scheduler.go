package pipeline

import (
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/taskctl/taskctl/internal/utils"

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

				err := s.run(stage)
				if err != nil {
					logrus.Errorf("stage %s failed: %v", stage.Name, err)
					stage.UpdateStatus(StatusError)
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

func (s *PipelineScheduler) run(stage *Stage) error {
	if stage.Pipeline != nil {
		return s.Schedule(stage.Pipeline)
	}

	return s.taskRunner.Run(stage.Task, stage.Variables, stage.Env)
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
