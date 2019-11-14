package runner

import (
	"context"
	"errors"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/task"
	"os"
	"os/exec"
	"time"
)

type TaskRunner struct {
	contexts map[string]*Context
	env      []string

	output *taskOutput
	ctx    context.Context
	cancel context.CancelFunc
}

func NewTaskRunner(contexts map[string]*Context, env []string, raw, quiet bool) *TaskRunner {
	tr := &TaskRunner{
		contexts: contexts,
		output:   NewTaskOutput(raw, quiet),
		env:      env,
	}

	tr.ctx, tr.cancel = context.WithCancel(context.Background())

	return tr
}

func (r *TaskRunner) Run(t *task.Task) (err error) {
	return r.RunWithEnv(t, r.env)
}

func (r *TaskRunner) RunWithEnv(t *task.Task, env []string) (err error) {
	c, ok := r.contexts[t.Context]
	if !ok {
		return errors.New("unknown context")
	}

	env = append(env, c.Env...)
	env = append(env, t.Env...)

	cwd := t.Dir
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			logrus.Fatalln(err)
		}
	}

	t.Start = time.Now()
	fmt.Println(aurora.Sprintf(aurora.Green("Running %s..."), aurora.Green(t.Name)))

	exargs := c.Executable.Args
	for _, command := range t.Command {
		args := append(exargs, command)

		cmd := exec.Command(c.Executable.Bin, args...)
		cmd.Dir = cwd
		cmd.Env = env

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logrus.Error(err)
		}
		t.SetStdout(stdout)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			logrus.Error(err)
		}
		t.SetStderr(stderr)

		err = r.runCommand(t, cmd)
		if err != nil {
			t.UpdateStatus(task.STATUS_ERROR)
			t.End = time.Now()
			return err
		}
	}

	t.End = time.Now()
	t.UpdateStatus(task.STATUS_DONE)

	fmt.Println(aurora.Sprintf(aurora.Green("%s finished. Elapsed %s"), aurora.Green(t.Name), aurora.Yellow(t.Duration())))

	return nil
}

func (r *TaskRunner) runCommand(t *task.Task, cmd *exec.Cmd) error {
	var done = make(chan struct{})
	var killed = make(chan struct{})
	go r.waitForInterruption(*cmd, done, killed)

	var flushed = make(chan struct{})
	go r.output.Scan(t, done, flushed)

	logrus.Debugf("Executing %s\r\n", cmd.String())
	err := cmd.Start()
	if err != nil {
		<-flushed
		return err
	}

	err = cmd.Wait()
	if err != nil {
		<-flushed
		return err
	}

	close(done)
	<-killed
	<-flushed

	return nil
}

func (r *TaskRunner) waitForInterruption(cmd exec.Cmd, done chan struct{}, killed chan struct{}) {
	defer close(killed)

	select {
	case <-r.ctx.Done():
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
