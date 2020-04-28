package runner

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/taskctl/taskctl/pkg/output"

	"github.com/sirupsen/logrus"

	taskctx "github.com/taskctl/taskctl/pkg/context"
	"github.com/taskctl/taskctl/pkg/task"
	"github.com/taskctl/taskctl/pkg/util"
)

type TaskRunner struct {
	variables Variables
	contexts  map[string]*taskctx.ExecutionContext

	ctx        context.Context
	cancel     context.CancelFunc
	dryRun     bool
	taskOutput *output.TaskOutput
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, outputFlavor string, variables Variables) (*TaskRunner, error) {
	r := &TaskRunner{
		contexts:  contexts,
		variables: variables,
	}

	var err error
	r.taskOutput, err = output.NewTaskOutput(outputFlavor, true)
	if err != nil {
		return nil, err
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	return r, nil
}

func (r *TaskRunner) Run(t *task.Task) (err error) {
	return r.RunWithVariables(t, r.variables)
}

func (r *TaskRunner) RunWithVariables(t *task.Task, variables Variables) (err error) {
	c, err := r.contextForTask(t)
	if err != nil {
		return errors.New("unknown context")
	}

	err = c.Up()
	if err != nil {
		return err
	}

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

	variables = variables.With("TASK_NAME", t.Name)

	for _, variant := range variations {
		for _, command := range t.Command {
			if t.Timeout != nil {
				ctx, _ = context.WithTimeout(ctx, *t.Timeout)
			}

			cmd, err := c.CreateCommand(ctx, command)
			if err != nil {
				return err
			}

			cmd.Env = append(cmd.Env, util.ConvertEnv(variant)...)
			cmd.Env = append(cmd.Env, variables...)

			var e *exec.ExitError
			err = r.runCommand(t, cmd)
			if err != nil {
				if errors.As(err, &e) && t.AllowFailure {
					continue
				}
				t.Errored = true
				t.End = time.Now()
				return err
			}
		}
	}

	if len(t.After) > 0 {
		for _, command := range t.After {
			cmd, err := c.CreateCommand(context.Background(), command)
			if err != nil {
				return err
			}

			cmd.Env = append(cmd.Env, variables...)
			err = r.runCommand(t, cmd)
			if err != nil {
				logrus.Warn(err)
			}
		}
	}

	t.End = time.Now()

	err = c.After()
	if err != nil {
		return err
	}

	return nil
}

func (r *TaskRunner) runCommand(t *task.Task, cmd *exec.Cmd) (err error) {
	var done = make(chan struct{})
	var killed = make(chan struct{})
	go r.waitForInterruption(*cmd, done, killed)

	logrus.Debugf("Executing %s", cmd.String())
	if r.dryRun {
		return nil
	}

	if t.Dir != "" {
		cmd.Dir = t.Dir
	}

	cmd.Dir, err = t.Interpolate(cmd.Dir, r.variables)
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	var flushed = make(chan struct{})
	go r.taskOutput.Stream(t, stdout, stderr, flushed)

	err = cmd.Start()
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
			logrus.Debug(err)
			return
		}
		logrus.Debugf("Killed %s", cmd.String())
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

}

func (r *TaskRunner) Finish() {
	for _, c := range r.contexts {
		if c.ScheduledForCleanup {
			c.Down()
		}
	}
	r.taskOutput.Close()
}

func (r *TaskRunner) DryRun() {
	r.dryRun = true
}
