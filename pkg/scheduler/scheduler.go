package scheduler

import (
	"github.com/taskctl/taskctl/pkg/util"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/runner"
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

func (s *PipelineScheduler) Schedule(p *pipeline.Pipeline) error {
	s.startTimer()
	defer s.stopTimer()
	var wg = sync.WaitGroup{}

	for !s.isDone(p) {
		if atomic.LoadInt32(&s.cancelled) == 1 {
			break
		}

		for _, stage := range p.Nodes() {
			switch stage.ReadStatus() {
			case pipeline.StatusWaiting, pipeline.StatusScheduled:
			default:
				continue
			}

			if !checkStatus(p, stage) {
				continue
			}

			wg.Add(1)
			stage.UpdateStatus(pipeline.StatusRunning)
			go func(stage *pipeline.Stage) {
				defer func() {
					stage.End = time.Now()
					wg.Done()
				}()

				stage.Start = time.Now()

				err := p.ProvideOutput(stage)
				if err != nil {
					logrus.Error(err)
				}

				if stage.Pipeline != nil {
					err = s.Schedule(stage.Pipeline)
				} else {
					err = s.taskRunner.RunWithEnv(stage.Task, util.ConvertEnv(stage.Env))
				}

				if err != nil {
					logrus.Error(err)
					stage.UpdateStatus(pipeline.StatusError)
					if !stage.AllowFailure {
						s.Cancel()
					}

					return
				}

				stage.UpdateStatus(pipeline.StatusDone)
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

func (s *PipelineScheduler) isDone(p *pipeline.Pipeline) bool {
	for _, stage := range p.Nodes() {
		switch stage.ReadStatus() {
		case pipeline.StatusWaiting, pipeline.StatusScheduled, pipeline.StatusRunning:
			return false
		}
	}

	return true
}

func (s *PipelineScheduler) Finish() {
	s.taskRunner.Finish()
}

func checkStatus(p *pipeline.Pipeline, stage *pipeline.Stage) (ready bool) {
	ready = true
	for _, dep := range p.To(stage.Name) {
		depStage, err := p.Node(dep)
		if err != nil {
			logrus.Fatal(err)
		}

		switch depStage.ReadStatus() {
		case pipeline.StatusDone:
			continue
		case pipeline.StatusError:
			if !depStage.AllowFailure {
				ready = false
				stage.UpdateStatus(pipeline.StatusCanceled)
			}
		case pipeline.StatusCanceled:
			ready = false
			stage.UpdateStatus(pipeline.StatusCanceled)
		default:
			ready = false
		}
	}

	return ready
}
