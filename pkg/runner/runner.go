package runner

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/task"
	"os/exec"
	"time"
)

type TaskRunner struct {
	contexts map[string]*ExecutionContext
	env      []string

	output *taskOutput
	ctx    context.Context
	cancel context.CancelFunc
}

func NewTaskRunner(contexts map[string]*ExecutionContext, env []string, raw, quiet bool) *TaskRunner {
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
	c, err := r.contextForTask(t)
	if err != nil {
		return errors.New("unknown context")
	}

	c.Up()
	err = c.Before()
	if err != nil {
		return err
	}

	t.Start = time.Now()
	log.Infof("Running task %s...", t.Name)

	var ctx = context.Background()
	for _, command := range t.Command {
		if t.Timeout != nil {
			ctx, _ = context.WithTimeout(ctx, *t.Timeout)
		}

		cmd, err := c.createCommand(ctx, command)
		if err != nil {
			return err
		}

		cmd.Env = append(cmd.Env, env...)
		cmd.Env = append(cmd.Env, r.env...)

		if t.Dir != "" {
			cmd.Dir = t.Dir
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		t.SetStdout(stdout)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		t.SetStderr(stderr)

		var e *exec.ExitError
		err = r.runCommand(t, cmd)
		if err != nil && !errors.As(err, &e) && !t.AllowFailure {
			t.End = time.Now()
			return err
		}
	}

	t.End = time.Now()

	err = c.After()
	if err != nil {
		return err
	}

	log.Infof("%s finished. Duration %s", t.Name, t.Duration())

	return nil
}

func (r *TaskRunner) runCommand(t *task.Task, cmd *exec.Cmd) error {
	var done = make(chan struct{})
	var killed = make(chan struct{})
	go r.waitForInterruption(*cmd, done, killed)

	var flushed = make(chan struct{})
	go r.output.Scan(t, flushed)

	log.Debugf("Executing %s", cmd.String())
	err := cmd.Start()
	if err != nil {
		close(done)
		<-flushed
		return err
	}

	<-flushed
	err = cmd.Wait()
	if err != nil {
		close(done)
		return err
	}

	close(done)
	<-killed

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
			log.Debug(err)
			return
		}
		log.Debugf("Killed %s", cmd.String())
		return
	case <-done:
		return
	}
}

func (r *TaskRunner) Cancel() {
	r.cancel()
}

func (r *TaskRunner) contextForTask(t *task.Task) (c *ExecutionContext, err error) {
	c, ok := r.contexts[t.Context]
	if !ok {
		return nil, errors.New("no such context")
	}

	if len(t.Env) > 0 {
		c, err = c.WithEnvs(t.Env)
		if err != nil {
			return nil, err
		}
	}

	c.ScheduleForCleanup()

	return c, nil
}

func (r *TaskRunner) DownContexts() {
	for _, c := range r.contexts {
		if c.scheduledForCleanup {
			c.Down()
		}
	}
}
