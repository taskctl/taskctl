package runner

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	taskctx "github.com/trntv/wilson/pkg/context"
	"github.com/trntv/wilson/pkg/task"
	"github.com/trntv/wilson/pkg/util"
	"os/exec"
	"time"
)

type TaskRunner struct {
	contexts map[string]*taskctx.ExecutionContext
	env      []string

	output *taskOutput
	ctx    context.Context
	cancel context.CancelFunc
	dryRun bool
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, env []string, raw, quiet, dryRun bool) *TaskRunner {
	tr := &TaskRunner{
		contexts: contexts,
		output:   NewTaskOutput(raw, quiet),
		env:      env,
		dryRun:   dryRun,
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

	var ctx = context.Background()

	var variations []map[string]string
	if len(t.Variations) > 0 {
		variations = t.Variations
	} else {
		variations = make([]map[string]string, 1)
	}

	for i, variant := range variations {
		log.Infof("Running task %s, variant %d...", t.Name, i+1)

		for _, command := range t.Command {
			if t.Timeout != nil {
				ctx, _ = context.WithTimeout(ctx, *t.Timeout)
			}

			cmd, err := c.CreateCommand(ctx, command)
			if err != nil {
				return err
			}

			cmd.Env = append(cmd.Env, util.ConvertEnv(variant)...)
			cmd.Env = append(cmd.Env, env...)
			cmd.Env = append(cmd.Env, r.env...)
			cmd.Env = append(cmd.Env, fmt.Sprintf("WI_TASK_NAME=%s", t.Name))

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
			if err != nil {
				if errors.As(err, &e) && t.AllowFailure {
					continue
				}
				t.End = time.Now()
				return err
			}
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
	if r.dryRun {
		return nil
	}

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

func (r *TaskRunner) contextForTask(t *task.Task) (c *taskctx.ExecutionContext, err error) {
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
		if c.ScheduledForCleanup {
			c.Down()
		}
	}
}
