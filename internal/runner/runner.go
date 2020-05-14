package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/taskctl/taskctl/internal/variables"

	"github.com/taskctl/taskctl/internal/output"

	"github.com/sirupsen/logrus"

	taskctx "github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/task"
	"github.com/taskctl/taskctl/internal/utils"
)

type Runner interface {
	Run(t *task.Task) error
	Cancel()
	Finish()
}

type TaskRunner struct {
	contexts  map[string]*taskctx.ExecutionContext
	variables variables.Container
	env       variables.Container
	dryRun    bool

	ctx        context.Context
	cancelFunc context.CancelFunc

	Stdout, Stderr io.Writer
	OutputFormat   string

	cleanupMutex sync.Mutex
	cleanupList  []*taskctx.ExecutionContext
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, vars variables.Container) (*TaskRunner, error) {
	r := &TaskRunner{
		OutputFormat: output.OutputFormatRaw,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		contexts:     contexts,
		variables:    vars,
		cleanupList:  make([]*taskctx.ExecutionContext, 0),
	}

	r.env = variables.NewVariablesFromEnv(os.Environ())

	if r.variables == nil {
		r.variables = &variables.Variables{}
	}

	r.ctx, r.cancelFunc = context.WithCancel(context.Background())

	return r, nil
}

func (r *TaskRunner) Run(t *task.Task) error {
	execContext, err := r.contextForTask(t)
	if err != nil {
		return err
	}

	err = execContext.Up()
	if err != nil {
		return err
	}

	err = execContext.Before()
	if err != nil {
		return err
	}

	defer func() {
		err := execContext.After()
		if err != nil {
			logrus.Error(err)
		}
	}()

	vars := r.variables.Merge(t.Variables)

	env := r.env.Merge(execContext.Env)
	env = env.With("TASK_NAME", t.Name)
	env = env.With("ARGS", vars.Get("Args"))
	env = env.Merge(t.Env)

	variations := t.Variations
	if variations == nil {
		variations = make([]map[string]string, 1)
	}

	if t.Dir != "" {
		t.Dir, err = utils.RenderString(t.Dir, vars.Map())
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

	outputFormat := r.OutputFormat
	if t.Interactive {
		outputFormat = output.OutputFormatRaw
	}

	taskOutput, err := output.NewTaskOutput(t, outputFormat, r.Stdout, r.Stderr)
	if err != nil {
		return err
	}

	err = taskOutput.Start()
	if err != nil {
		return err
	}

	t.Start = time.Now()
	for _, variant := range variations {
		for _, command := range t.Command {
			command, err = utils.RenderString(command, t.Variables.Merge(vars).Map())
			if err != nil {
				return err
			}

			ctx := r.ctx
			var cancelFunc context.CancelFunc
			if t.Timeout != nil {
				ctx, cancelFunc = context.WithTimeout(r.ctx, *t.Timeout)
			}

			cmd, err := createCommand(
				ctx,
				execContext,
				command,
				env,
				variables.NewVariables(variant),
			)
			if err != nil {
				if cancelFunc != nil {
					cancelFunc()
				}
				return err
			}

			if t.Dir != "" {
				cmd.Dir = t.Dir
			}

			cmd.Stdout = taskOutput.Stdout()
			cmd.Stderr = taskOutput.Stderr()

			if t.Interactive {
				cmd.Stdin = os.Stdin
			}

			cmdOutput, err := r.executeCommand(t, cmd)
			if cancelFunc != nil {
				cancelFunc()
			}

			if err != nil {
				logrus.Debug(err.Error())
				if utils.IsExitError(err) && t.AllowFailure {
					continue
				}
				t.Errored = true
				t.Error = err
				break
			}

			vars = vars.With("Output", string(cmdOutput))
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
			cmd, err := createCommand(r.ctx, execContext, command)
			if err != nil {
				return fmt.Errorf("\"after\" command failed: %w", err)
			}

			if t.Dir != "" {
				cmd.Dir = t.Dir
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

func (r *TaskRunner) ScheduleForCleanup(c *taskctx.ExecutionContext) {
	r.cleanupMutex.Lock()
	defer r.cleanupMutex.Unlock()

	r.cleanupList = append(r.cleanupList, c)
}

func (r *TaskRunner) executeCommand(t *task.Task, cmd *exec.Cmd) ([]byte, error) {
	logrus.Debugf("Executing %s", cmd.String())
	if r.dryRun {
		return nil, nil
	}

	var buf bytes.Buffer

	cmd.Stdout = io.MultiWriter(&buf, cmd.Stdout)
	cmd.Stderr = io.MultiWriter(&buf, cmd.Stderr)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	t.ExitCode = cmd.ProcessState.ExitCode()
	logrus.Debugf("Executed %s", cmd.String())

	return buf.Bytes(), err
}

func (r *TaskRunner) contextForTask(t *task.Task) (c *taskctx.ExecutionContext, err error) {
	if t.Context == "" {
		return taskctx.DefaultContext(), nil
	}

	c, ok := r.contexts[t.Context]
	if !ok {
		return nil, fmt.Errorf("no such context %s", t.Context)
	}

	r.cleanupList = append(r.cleanupList, c)

	return c, nil
}

func (r *TaskRunner) checkTaskCondition(t *task.Task) (bool, error) {
	c, err := r.contextForTask(t)
	if err != nil {
		return false, err
	}

	cmd, err := createCommand(r.ctx, c, t.Condition)
	if err != nil {
		return false, err
	}

	if t.Dir != "" {
		cmd.Dir = t.Dir
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
