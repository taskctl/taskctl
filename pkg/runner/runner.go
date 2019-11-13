package runner

import (
	"context"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/task"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

const LOOP_DELAY = 50 * time.Millisecond

type PipelineRunner struct {
	pipeline *task.Pipeline
	contexts map[string]*Context
	output   *taskOutput

	rctx      context.Context
	cancel    context.CancelFunc
	cancelled int32
	wg        sync.WaitGroup
	pause     time.Duration

	Start time.Time
	End   time.Time
}

func NewRunner(pipeline *task.Pipeline, contexts map[string]*Context, raw bool, quiet bool) *PipelineRunner {

	r := &PipelineRunner{
		pipeline: pipeline,
		contexts: contexts,
		output:   NewTaskOutput(raw, quiet),
	}

	r.rctx, r.cancel = context.WithCancel(context.Background())

	return r
}

func (pr *PipelineRunner) Schedule() {
	pr.startTimer()
	defer pr.stopTimer()

	var done = false
	for {
		if done {
			break
		}

		if atomic.LoadInt32(&pr.cancelled) == 1 {
			break
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
				go pr.Run(t)
			}
		}

		time.Sleep(LOOP_DELAY)
	}

	pr.wg.Wait()
}

func (pr *PipelineRunner) Run(t *task.Task) {
	pr.wg.Add(1)
	defer pr.wg.Done()

	c := pr.contexts[t.Context]
	var err error
	var env = append(c.Env, t.Env...)

	cwd := t.Dir
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	t.Start = time.Now()
	if !t.SwapStatus(task.STATUS_WAITING, task.STATUS_RUNNING) {
		logrus.Fatal("unexpected task status")
	}
	fmt.Println(aurora.Sprintf(aurora.Green("Running %s..."), aurora.Green(t.Name)))

	for _, command := range t.Command {
		if t.ReadStatus() == task.STATUS_ERROR {
			break
		}

		args := append(c.Executable.Args, command)

		cmd := exec.Command(c.Executable.Bin, args...)
		cmd.Dir = cwd
		cmd.Env = env

		t.Stdout, err = cmd.StdoutPipe()
		if err != nil {
			logrus.Error(err)
		}

		t.Stderr, err = cmd.StderrPipe()
		if err != nil {
			logrus.Error(err)
		}

		pr.runCommand(t, cmd)
	}

	t.End = time.Now()
	if t.ReadStatus() == task.STATUS_ERROR {
		pr.Cancel()
		return
	}

	t.UpdateStatus(task.STATUS_DONE)

	fmt.Println(aurora.Sprintf(aurora.Green("%s finished. Elapsed %s"), aurora.Green(t.Name), aurora.Yellow(t.Duration())))
}

func (pr *PipelineRunner) Cancel() {
	atomic.StoreInt32(&pr.cancelled, 1)
	pr.cancel()
}

func (pr *PipelineRunner) runCommand(t *task.Task, cmd *exec.Cmd) {
	var done = make(chan struct{})
	var killed = make(chan struct{})
	go pr.waitForInterruption(*cmd, done, killed)

	var flushed = make(chan struct{})
	go pr.output.Scan(t, done, flushed)

	logrus.Debugf("Executing %s\r\n", cmd.String())
	err := cmd.Start()
	if err != nil {
		t.UpdateStatus(task.STATUS_ERROR)
		<-flushed
		pr.Cancel()
		return
	}

	err = cmd.Wait()
	if err != nil {
		t.UpdateStatus(task.STATUS_ERROR)
		<-flushed
		pr.Cancel()
		return
	}

	close(done)
	<-killed
	<-flushed
}

func (pr *PipelineRunner) startTimer() {
	pr.Start = time.Now()
}
func (pr *PipelineRunner) stopTimer() {
	pr.End = time.Now()
}

func (pr *PipelineRunner) waitForInterruption(cmd exec.Cmd, done chan struct{}, killed chan struct{}) {
	defer close(killed)

	select {
	case <-pr.rctx.Done():
		if cmd.ProcessState == nil || cmd.ProcessState.Exited() {
			return
		}
		if err := cmd.Process.Kill(); err != nil {
			logrus.Debug(err)
			return
		}
		logrus.Debugf("Killed %s", cmd.String())
		return
	case <-done:
		return
	}
}
