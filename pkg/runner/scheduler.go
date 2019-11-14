package runner

import (
	"github.com/trntv/wilson/pkg/task"
	"sync"
	"sync/atomic"
	"time"
)

type PipelineScheduler struct {
	pipeline   *task.Pipeline
	taskRunner *TaskRunner
	pause      time.Duration

	cancelled int32
	wg        sync.WaitGroup

	Start time.Time
	End   time.Time
}

func NewScheduler(pipeline *task.Pipeline, contexts map[string]*Context, raw, quiet bool) *PipelineScheduler {
	r := &PipelineScheduler{
		pipeline:   pipeline,
		pause:      50 * time.Millisecond,
		taskRunner: NewTaskRunner(contexts, raw, quiet),
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
		for _, t := range s.pipeline.Nodes() {
			switch t.ReadStatus() {
			case task.STATUS_WAITING, task.STATUS_SCHEDULED:
				done = false
			case task.STATUS_RUNNING:
				done = false
				continue
			default:
				continue
			}

			var ready = true
			for _, dep := range s.pipeline.To(t.Name) {
				depTask := s.pipeline.Node(dep)
				switch depTask.ReadStatus() {
				case task.STATUS_DONE:
					continue
				case task.STATUS_ERROR, task.STATUS_CANCELED:
					ready = false
					t.UpdateStatus(task.STATUS_CANCELED)
				default:
					ready = false
				}
			}

			if ready {
				s.wg.Add(1)
				t.UpdateStatus(task.STATUS_RUNNING)
				go func(t *task.Task) {
					defer s.wg.Done()
					err := s.taskRunner.Run(t)
					if err != nil {
						t.UpdateStatus(task.STATUS_ERROR)
						s.Cancel()
					} else {
						t.UpdateStatus(task.STATUS_DONE)
					}
				}(t)
			}
		}

		time.Sleep(s.pause)
	}

	s.wg.Wait()
}

func (s *PipelineScheduler) startTimer() {
	s.Start = time.Now()
}

func (s *PipelineScheduler) stopTimer() {
	s.End = time.Now()
}

func (s *PipelineScheduler) Cancel() {
	atomic.StoreInt32(&s.cancelled, 1)
	s.taskRunner.cancel()
}
