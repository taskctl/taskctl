package scheduler

import (
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/task"
	"sync"
	"sync/atomic"
	"time"
)

type PipelineScheduler struct {
	pipeline   *Pipeline
	taskRunner *runner.TaskRunner
	pause      time.Duration

	cancelled int32
	wg        sync.WaitGroup

	Start time.Time
	End   time.Time
}

func NewScheduler(pipeline *Pipeline, contexts map[string]*runner.ExecutionContext, env []string, raw, quiet bool) *PipelineScheduler {
	r := &PipelineScheduler{
		pipeline:   pipeline,
		pause:      50 * time.Millisecond,
		taskRunner: runner.NewTaskRunner(contexts, env, raw, quiet),
	}

	return r
}

func (s *PipelineScheduler) Schedule() {
	s.startTimer()
	defer s.stopTimer()

	var done = false
	for {
		if done {
			break
		}

		if atomic.LoadInt32(&s.cancelled) == 1 {
			break
		}

		done = true
		for name, stage := range s.pipeline.Nodes() {
			switch stage.Task.ReadStatus() {
			case task.STATUS_WAITING, task.STATUS_SCHEDULED:
				done = false
			case task.STATUS_RUNNING:
				done = false
				continue
			default:
				continue
			}

			var ready = true
			for _, dep := range s.pipeline.To(name) {
				depStage := s.pipeline.Node(dep)
				switch depStage.Task.ReadStatus() {
				case task.STATUS_DONE:
					continue
				case task.STATUS_ERROR, task.STATUS_CANCELED:
					ready = false
					stage.Task.UpdateStatus(task.STATUS_CANCELED)
				default:
					ready = false
				}
			}

			if ready {
				s.wg.Add(1)
				stage.Task.UpdateStatus(task.STATUS_RUNNING)
				go func(t *task.Task, env []string) {
					defer s.wg.Done()
					err := s.taskRunner.RunWithEnv(t, env)
					if err != nil {
						log.Error(err)
						t.UpdateStatus(task.STATUS_ERROR)
						s.Cancel()
					} else {
						t.UpdateStatus(task.STATUS_DONE)
					}
				}(&stage.Task, s.pipeline.env[name])
			}
		}

		time.Sleep(s.pause)
	}

	s.wg.Wait()
	s.taskRunner.DownContexts()
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
