package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/taskctl/taskctl/internal/output"

	"github.com/sirupsen/logrus"

	taskctx "github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/utils"
)

type TaskRunner struct {
	contexts  map[string]*taskctx.ExecutionContext
	variables *utils.Variables
	env       *utils.Variables
	dryRun    bool

	ctx        context.Context
	cancelFunc context.CancelFunc

	executor     CommandExecutor
	outputFormat string

	cleanupMutex sync.Mutex
	cleanupList  []*taskctx.ExecutionContext
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, outputFormat string, variables *utils.Variables) (*TaskRunner, error) {
	r := &TaskRunner{
		contexts:    contexts,
		variables:   variables,
		env:         &utils.Variables{},
		cleanupList: make([]*taskctx.ExecutionContext, 0),
		executor:    NewDefaultCommandExecutor(),
	}

	r.outputFormat = outputFormat
	r.ctx, r.cancelFunc = context.WithCancel(context.Background())

	return r, nil
}

func (r *TaskRunner) Command(c *taskctx.ExecutionContext, ctx context.Context, command string, t *task.Task) (*exec.Cmd, error) {
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

func (r *TaskRunner) Run(t *task.Task, variables *utils.Variables, env *utils.Variables) error {
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
		err := c.After()
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

	outputFormat := r.outputFormat
	if t.Interactive {
		outputFormat = output.OutputFormatRaw
	}

	taskOutput, err := output.NewTaskOutput(t, outputFormat)
	if err != nil {
		return err
	}

	err = taskOutput.Start()
	if err != nil {
		return err
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

			ctx := r.ctx
			var cancelFunc context.CancelFunc
			if t.Timeout != nil {
				ctx, cancelFunc = context.WithTimeout(r.ctx, *t.Timeout)
			}

			cmd, err := r.Command(c, ctx, command, t)
			if err != nil {
				if cancelFunc != nil {
					cancelFunc()
				}
				return err
			}

			cmd.Stdout = taskOutput.Stdout()
			cmd.Stderr = taskOutput.Stderr()

			if t.Interactive {
				cmd.Stdin = os.Stdin
			}

			cmd.Env = append(cmd.Env, utils.ConvertEnv(variant)...)
			cmd.Env = append(cmd.Env, utils.ConvertEnv(env.Map())...)

			cmdOutput, err = r.executeCommand(t, cmd)
			if cancelFunc != nil {
				cancelFunc()
			}

			if err != nil {
				if utils.IsExitError(err) && t.AllowFailure {
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

	err = taskOutput.Finish()
	if err != nil {
		logrus.Warning(err)
	}

	r.storeOutput(t)

	if t.Errored {
		return t.Error
	}

	if len(t.After) > 0 {
		for _, command := range t.After {
			cmd, err := r.Command(c, r.ctx, command, t)
			if err != nil {
				return fmt.Errorf("\"after\" command failed: %w", err)
			}

			cmd.Env = append(cmd.Env, utils.ConvertEnv(env.Map())...)
			_, err = r.executeCommand(t, cmd)
			if err != nil {
				logrus.Warn(err)
			}
		}
	}

	return nil
}

func (r *TaskRunner) Cancel() {
	logrus.Debug("Runner has been cancelled")
	r.cancelFunc()
}

func (r *TaskRunner) Finish() {
	for _, c := range r.cleanupList {
		c.Down()
	}
	output.Close()
}

func (r *TaskRunner) DryRun() {
	r.dryRun = true
}

func (r *TaskRunner) executeCommand(t *task.Task, cmd *exec.Cmd) (string, error) {
	logrus.Debugf("Executing %s", cmd.String())
	if r.dryRun {
		return "", nil
	}

	b, err := r.executor.Execute(cmd)
	t.ExitCode = cmd.ProcessState.ExitCode()
	logrus.Debugf("Executed %s", cmd.String())

	return string(b), err
}

func (r *TaskRunner) contextForTask(t *task.Task) (c *taskctx.ExecutionContext, err error) {
	c, ok := r.contexts[t.Context]
	if !ok {
		return nil, errors.New("no such context")
	}

	r.cleanupList = append(r.cleanupList, c)

	return c, nil
}

func (r *TaskRunner) checkTaskCondition(t *task.Task) (bool, error) {
	c, err := r.contextForTask(t)
	if err != nil {
		return false, err
	}

	cmd, err := r.Command(c, r.ctx, t.Condition, t)
	if err != nil {
		return false, err
	}

	_, err = r.executeCommand(t, cmd)
	if err != nil {
		if utils.IsExitError(err) {
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
