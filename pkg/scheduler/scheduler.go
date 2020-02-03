package scheduler

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/pkg/context"
	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/runner"
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

func NewScheduler(contexts map[string]*context.ExecutionContext, env []string, raw, quiet, dryRun bool) *PipelineScheduler {
	r := &PipelineScheduler{
		pause:      50 * time.Millisecond,
		taskRunner: runner.NewTaskRunner(contexts, env, raw, quiet, dryRun),
	}

	return r
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

			if !s.checkStatus(p, stage) {
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
				var err error
				if stage.Pipeline != nil {
					err = s.Schedule(stage.Pipeline)
					fmt.Print("test")
				} else {
					err = s.taskRunner.RunWithEnv(stage.Task, p.Env[stage.Name])
				}
				if err != nil {
					log.Error(err)
					stage.UpdateStatus(pipeline.StatusError)
					if !stage.AllowFailure {
						s.Cancel()
					}
				} else {
					stage.UpdateStatus(pipeline.StatusDone)
				}
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

func (s *PipelineScheduler) DownContexts() {
	s.taskRunner.DownContexts()
}

func (s *PipelineScheduler) checkStatus(p *pipeline.Pipeline, stage *pipeline.Stage) (ready bool) {
	ready = true
	for _, dep := range p.To(stage.Name) {
		depStage, err := p.Node(dep)
		if err != nil {
			log.Fatal(err)
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
