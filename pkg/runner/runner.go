package runner

import (
	"context"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/task"
	"os"
	"os/exec"
	"time"
)

const LOOP_DELAY = 50 * time.Millisecond

type PipelineRunner struct {
	pipeline     *task.Pipeline
	contexts     map[string]*Context

	rctx context.Context
	cancel context.CancelFunc
	cancelled bool

	Start time.Time
	End time.Time
}

func NewRunner(pipeline *task.Pipeline, contexts map[string]*Context) *PipelineRunner {

	r := &PipelineRunner{
		pipeline: pipeline,
		contexts: contexts,
	}

	r.rctx, r.cancel = context.WithCancel(context.Background())

	return r
}

func (pr *PipelineRunner) Run() {
	pr.startTimer()
	defer pr.stopTimer()

	var done = false
	for {

		if done || pr.cancelled {
			return
		}

		done = true
		for _, t := range pr.pipeline.Nodes() {
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
			for _, dep := range pr.pipeline.To(t.Name) {
				depTask := pr.pipeline.Node(dep)
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
				if !t.SwapStatus(task.STATUS_WAITING, task.STATUS_RUNNING) {
					logrus.Fatal("Context: unexpected task status")
				}
				go pr.RunTask(t)
			}
		}

		time.Sleep(LOOP_DELAY)
	}
}

func (pr *PipelineRunner) RunTask(t *task.Task) {
	fmt.Println(aurora.Sprintf(aurora.Green("Starting %s..."), aurora.Green(t.Name)))
	c := pr.contexts[t.Context]
	var err error
	var env = append(c.Env, t.Env...)
	t.Start = time.Now()
	for i, command := range t.Command {
		args := append(c.Executable.Args, command)

		cmd := exec.Command(c.Executable.Bin, args...)
		cmd.Dir, err = os.Getwd()
		if err != nil {
			logrus.Fatalln(err)
		}

		logrus.Debugf("Executing %s\r\n", cmd.String())

		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Start()
		if err != nil {
			t.UpdateStatus(task.STATUS_ERROR)
			break
		}

		var finished = make(chan struct{})
		go func(finished chan struct{}, name string, index int) {
			select {
			case <-pr.rctx.Done():
				err := cmd.Process.Kill()
				if err != nil {
					logrus.Error(err)
				}
				logrus.Debugf("Killed %s#%d", name, index)
				return
			case <-finished:
				return
			}
		}(finished, t.Name, i)

		err = cmd.Wait()
		close(finished)
		if err != nil {
			t.UpdateStatus(task.STATUS_ERROR)
			pr.Cancel()
			break
		}
	}

	if t.ReadStatus() != task.STATUS_ERROR {
		t.UpdateStatus(task.STATUS_DONE)
	}

	t.End = time.Now()
	fmt.Println(aurora.Sprintf(aurora.Green("%s finished. Elapsed %s"), aurora.Green(t.Name), aurora.Yellow(t.Duration())))
}

func (pr *PipelineRunner) Cancel() {
	pr.cancel()
	pr.cancelled = true
}

func (pr *PipelineRunner) startTimer() {
	pr.Start = time.Now()
}
func (pr *PipelineRunner) stopTimer() {
	pr.End = time.Now()
}

