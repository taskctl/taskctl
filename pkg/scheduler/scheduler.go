package scheduler

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/runner"
	"sync"
	"sync/atomic"
	"time"
)

type PipelineScheduler struct {
	taskRunner *runner.TaskRunner
	pause      time.Duration

	cancelled int32

	Start time.Time
	End   time.Time
}

func NewScheduler(contexts map[string]*runner.ExecutionContext, env []string, raw, quiet bool) *PipelineScheduler {
	r := &PipelineScheduler{
		pause:      50 * time.Millisecond,
		taskRunner: runner.NewTaskRunner(contexts, env, raw, quiet),
	}

	return r
}

func (s *PipelineScheduler) Schedule(pipeline *Pipeline) error {
	s.startTimer()
	defer s.stopTimer()
	var wg = sync.WaitGroup{}

	for !s.isDone(pipeline) {
		if atomic.LoadInt32(&s.cancelled) == 1 {
			break
		}

		for _, stage := range pipeline.Nodes() {
			switch stage.ReadStatus() {
			case StatusWaiting, StatusScheduled:
			default:
				continue
			}

			if !s.checkStatus(pipeline, stage) {
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
					fmt.Print("test")
				} else {
					err = s.taskRunner.RunWithEnv(stage.Task, pipeline.env[stage.Name])
				}
				if err != nil {
					log.Error(err)
					stage.UpdateStatus(StatusError)
					if !stage.AllowFailure {
						s.Cancel()
					}
				} else {
					stage.UpdateStatus(StatusDone)
				}
			}(stage)
		}

		time.Sleep(s.pause)
	}

	wg.Wait()

	return pipeline.error
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

func (s *PipelineScheduler) isDone(pipeline *Pipeline) bool {
	for _, stage := range pipeline.Nodes() {
		switch stage.ReadStatus() {
		case StatusWaiting, StatusScheduled, StatusRunning:
			return false
		}
	}

	return true
}

func (s *PipelineScheduler) DownContexts() {
	s.taskRunner.DownContexts()
}

func (s *PipelineScheduler) checkStatus(pipeline *Pipeline, stage *Stage) (ready bool) {
	ready = true
	for _, dep := range pipeline.To(stage.Name) {
		depStage, err := pipeline.Node(dep)
		if err != nil {
			log.Fatal(err)
		}

		switch depStage.ReadStatus() {
		case StatusDone:
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
