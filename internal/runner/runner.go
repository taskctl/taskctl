package runner

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/taskctl/taskctl/internal/output"

	"github.com/sirupsen/logrus"

	taskctx "github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/util"
)

type TaskRunner struct {
	contexts  map[string]*taskctx.ExecutionContext
	variables *util.Variables
	env       *util.Variables

	ctx        context.Context
	cancel     context.CancelFunc
	dryRun     bool
	taskOutput *output.TaskOutput

	cleanupMutex sync.Mutex
	cleanupList  []*taskctx.ExecutionContext
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, outputFormat string, variables *util.Variables) (*TaskRunner, error) {
	r := &TaskRunner{
		contexts:    contexts,
		variables:   variables,
		env:         &util.Variables{},
		cleanupList: make([]*taskctx.ExecutionContext, 0),
	}

	var err error
	r.taskOutput, err = output.NewTaskOutput(outputFormat)
	if err != nil {
		return nil, err
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())

	return r, nil
}

func (r *TaskRunner) Run(t *task.Task, variables *util.Variables, env *util.Variables) (err error) {
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

	env = env.Merge(r.env)
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
			logrus.Infof("task %s was skipped", t.Name)
			t.Skipped = true
			return nil
		}
	}

	err = r.taskOutput.Start(t)
	if err != nil {
		logrus.Warning(err)
	}

	t.Start = time.Now()
	for _, variant := range t.Variations {
		var cmdOutput string
		for _, command := range t.Command {
			variables = variables.With("Output", cmdOutput)

			command, err = t.Interpolate(command, variables)
			if err != nil {
				return err
			}

			ctx, cancelFn := context.WithCancel(r.ctx)
			if t.Timeout != nil {
				ctx, cancelFn = context.WithTimeout(ctx, *t.Timeout)
			}

			cmd, err := r.BuildCommand(c, ctx, command, t)
			if err != nil {
				cancelFn()
				return err
			}

			cmd.Env = append(cmd.Env, util.ConvertEnv(variant)...)
			cmd.Env = append(cmd.Env, util.ConvertEnv(env.Map())...)

			cmdOutput, err = r.executeCommand(t, cmd)
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

	r.storeOutput(t)

	if t.Errored {
		return t.Error
	}

	if len(t.After) > 0 {
		for _, command := range t.After {
			cmd, err := r.BuildCommand(c, r.ctx, command, t)
			if err != nil {
				return err
			}

			cmd.Env = append(cmd.Env, util.ConvertEnv(env.Map())...)
			_, err = r.executeCommand(t, cmd)
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
	for _, c := range r.cleanupList {
		c.Down()
	}
	r.taskOutput.Close()
}

func (r *TaskRunner) DryRun() {
	r.dryRun = true
}

func (r *TaskRunner) executeCommand(t *task.Task, cmd *exec.Cmd) (output string, err error) {
	var done = make(chan struct{})
	var killed = make(chan struct{})
	go r.waitForInterruption(*cmd, done, killed)

	logrus.Debugf("Executing %s", cmd.String())
	if r.dryRun {
		return output, nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return output, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return output, err
	}

	var flushed = make(chan []byte)
	go r.taskOutput.Stream(t, stdout, stderr, flushed)

	err = cmd.Start()
	if err != nil {
		close(done)
		<-flushed
		return output, err
	}

	buf := <-flushed
	err = cmd.Wait()
	if err != nil {
		close(done)
		return output, err
	}

	close(done)
	<-killed

	return string(buf), nil
}

func (r *TaskRunner) BuildCommand(c *taskctx.ExecutionContext, ctx context.Context, command string, t *task.Task) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, c.Executable.Bin, c.Executable.Args...)
	cmd.Args = append(cmd.Args, command)
	cmd.Env = c.Env
	cmd.Dir = c.Dir

	if cmd == nil {
		return nil, errors.New("failed to build command")
	}

	if t != nil && t.Dir != "" {
		cmd.Dir = t.Dir
	}

	return cmd, nil
}

func (r *TaskRunner) contextForTask(t *task.Task) (c *taskctx.ExecutionContext, err error) {
	c, ok := r.contexts[t.Context]
	if !ok {
		return nil, errors.New("no such context")
	}

	r.cleanupList = append(r.cleanupList, c)

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

	cmd, err := r.BuildCommand(c, r.ctx, t.Condition, t)
	if err != nil {
		return false, err
	}

	_, err = r.executeCommand(t, cmd)
	if err != nil {
		var e *exec.ExitError
		if errors.As(err, &e) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *TaskRunner) storeOutput(t *task.Task) {
	var envVarName string
	varName := fmt.Sprintf("Tasks.%s.Output", strings.Title(t.Name))

	if t.ExportAs == "" {
		envVarName = fmt.Sprintf("%s_OUTPUT", strings.ToUpper(t.Name))
		envVarName = regexp.MustCompile("[^a-zA-Z0-9_]").ReplaceAllString(envVarName, "_")
	} else {
		envVarName = t.ExportAs
	}

	r.env.Set(envVarName, t.Log.Stdout.String())
	r.variables.Set(varName, t.Log.Stdout.String())
}

func (r *TaskRunner) ScheduleForCleanup(c *taskctx.ExecutionContext) {
	r.cleanupMutex.Lock()
	defer r.cleanupMutex.Unlock()

	r.cleanupList = append(r.cleanupList, c)
}
