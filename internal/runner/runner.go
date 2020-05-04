package runner

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/taskctl/taskctl/internal/config"

	"github.com/taskctl/taskctl/internal/output"

	"github.com/sirupsen/logrus"

	taskctx "github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/util"
)

type TaskRunner struct {
	contexts  map[string]*taskctx.ExecutionContext
	variables config.Variables

	ctx        context.Context
	cancel     context.CancelFunc
	dryRun     bool
	taskOutput *output.TaskOutput
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, outputFlavor string, variables config.Variables) (*TaskRunner, error) {
	r := &TaskRunner{
		contexts:  contexts,
		variables: variables,
	}

	var err error
	r.taskOutput, err = output.NewTaskOutput(outputFlavor)
	if err != nil {
		return nil, err
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	return r, nil
}

func (r *TaskRunner) Run(t *task.Task, variables config.Variables, env config.Variables) (err error) {
	c, err := r.contextForTask(t)
	if err != nil {
		return err
	}

	err = c.Up()
	if err != nil {
		return err
	}

	err = c.Before()
	if err != nil {
		return err
	}

	defer func() {
		err = c.After()
		if err != nil {
			logrus.Error(err)
		}
	}()

	env = env.With("TASK_NAME", t.Name)
	env = env.With("ARGS", variables.Get("Args"))
	env = env.Merge(t.Env)
	variables = r.variables.Merge(variables)

	if t.Dir != "" {
		t.Dir, err = t.Interpolate(t.Dir, variables)
		if err != nil {
			return err
		}
	}

	if t.Condition != "" {
		meets, err := r.checkTaskCondition(t)
		if err != nil {
			return err
		}

		if !meets {
			logrus.Infof("checkTaskCondition %s was skipped", t.Name)
			t.Skipped = true
			return nil
		}
	}

	err = r.taskOutput.Start(t)
	if err != nil {
		logrus.Warning(err)
	}

	ctx := context.Background()
	t.Start = time.Now()
	for _, variant := range t.Variations {
		for _, command := range t.Command {
			command, err = t.Interpolate(command, variables)
			if err != nil {
				return err
			}

			ctx, cancelFn := context.WithCancel(ctx)
			if t.Timeout != nil {
				ctx, cancelFn = context.WithTimeout(ctx, *t.Timeout)
			}

			cmd, err := c.BuildCommand(ctx, command, t)
			if err != nil {
				cancelFn()
				return err
			}

			cmd.Env = append(cmd.Env, util.ConvertEnv(variant)...)
			cmd.Env = append(cmd.Env, util.ConvertEnv(env)...)

			err = r.executeCommand(t, cmd)
			cancelFn()

			if err != nil {
				var e *exec.ExitError
				if errors.As(err, &e) && t.AllowFailure {
					continue
				}
				t.Errored = true
				t.Error = err
				break
			}
		}
		if t.Errored {
			break
		}
	}
	t.End = time.Now()
	err = r.taskOutput.Finish(t)
	if err != nil {
		logrus.Warning(err)
	}

	if t.Errored {
		return t.Error
	}

	if len(t.After) > 0 {
		for _, command := range t.After {
			cmd, err := c.BuildCommand(context.Background(), command, t)
			if err != nil {
				return err
			}

			cmd.Env = append(cmd.Env, util.ConvertEnv(variables)...)
			err = r.executeCommand(t, cmd)
			if err != nil {
				logrus.Warn(err)
			}
		}
	}

	return nil
}

func (r *TaskRunner) Cancel() {
	r.cancel()
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

func (r *TaskRunner) executeCommand(t *task.Task, cmd *exec.Cmd) (err error) {
	var done = make(chan struct{})
	var killed = make(chan struct{})
	go r.waitForInterruption(*cmd, done, killed)

	logrus.Debugf("Executing %s", cmd.String())
	if r.dryRun {
		return nil
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

func (r *TaskRunner) contextForTask(t *task.Task) (c *taskctx.ExecutionContext, err error) {
	c, ok := r.contexts[t.Context]
	if !ok {
		return nil, errors.New("no such context")
	}

	c.ScheduleForCleanup()

	return c, nil
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

func (r *TaskRunner) checkTaskCondition(t *task.Task) (bool, error) {
	c, err := r.contextForTask(t)
	if err != nil {
		return false, err
	}

	cmd, err := c.BuildCommand(r.ctx, t.Condition, t)
	if err != nil {
		return false, err
	}

	err = r.executeCommand(t, cmd)
	if err != nil {
		var e *exec.ExitError
		if errors.As(err, &e) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
