package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/taskctl/taskctl/internal/executor"

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
	Executor  executor.Executor
	DryRun    bool
	contexts  map[string]*taskctx.ExecutionContext
	variables variables.Container
	env       variables.Container

	ctx        context.Context
	cancelFunc context.CancelFunc

	Stdin          io.Reader
	Stdout, Stderr io.Writer
	OutputFormat   string

	cleanupList sync.Map
}

func NewTaskRunner(contexts map[string]*taskctx.ExecutionContext, vars variables.Container) (*TaskRunner, error) {
	exec, err := executor.NewDefaultExecutor()
	if err != nil {
		return nil, err
	}

	r := &TaskRunner{
		Executor:     exec,
		OutputFormat: output.OutputFormatRaw,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		contexts:     contexts,
		variables:    vars,
	}

	r.env = variables.NewVariables(map[string]string{"ARGS": vars.Get("Args")})

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
	env = env.Merge(t.Env)

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

	variations := make([]map[string]string, 1)
	if t.Variations != nil {
		variations = t.Variations
	}

	var stdin io.Reader
	if t.Interactive {
		stdin = r.Stdin
	}

	var job, prev *executor.Job
	for _, variant := range variations {
		for _, command := range t.Commands {
			j, err := r.Compile(
				command,
				t,
				execContext,
				stdin,
				taskOutput.Stdout(),
				taskOutput.Stderr(),
				env.Merge(variables.NewVariables(variant)),
				t.Variables.Merge(vars),
			)
			if err != nil {
				return err
			}

			if job == nil {
				job = j
			}

			if prev == nil {
				prev = j
			} else {
				prev.Next = j
				prev = prev.Next
			}
		}
	}

	err = taskOutput.Start()
	if err != nil {
		return err
	}

	t.Start = time.Now()
	var prevOutput []byte
	for nextJob := job; nextJob != nil; nextJob = nextJob.Next {
		var err error
		nextJob.Vars.Set("Output", string(prevOutput))

		prevOutput, err = r.Executor.Execute(r.ctx, nextJob)
		if err != nil {
			logrus.Debug(err.Error())
			if utils.IsExitError(err) && t.AllowFailure {
				continue
			}
			t.Errored = true
			t.Error = err
			break
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

	if t.Errored {
		return t.Error
	}

	r.storeTaskOutput(t)

	if len(t.After) > 0 {
		for _, command := range t.After {
			job, err := r.Compile(command, t, execContext, nil, r.Stdout, r.Stderr, env, vars)
			if err != nil {
				return fmt.Errorf("\"after\" Command failed: %w", err)
			}

			_, err = r.Executor.Execute(r.ctx, job)
			if err != nil {
				logrus.Warning(err)
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
	r.cleanupList.Range(func(key, value interface{}) bool {
		value.(*taskctx.ExecutionContext).Down()
		return true
	})
	output.Close()
}

func (r *TaskRunner) Compile(command string, t *task.Task, executionCtx *taskctx.ExecutionContext, stdin io.Reader, stdout, stderr io.Writer, env, vars variables.Container) (*executor.Job, error) {
	j := &executor.Job{
		Timeout: t.Timeout,
		Env:     env,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Vars:    vars,
	}

	c := make([]string, 0)
	if executionCtx.Executable != nil {
		c = append(c, executionCtx.Executable.Bin)
		c = append(c, executionCtx.Executable.Args...)
	}
	c = append(c, command)
	j.Command = strings.Join(c, " ")

	var err error
	j.Dir, err = r.resolveDir(t, executionCtx)
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (r *TaskRunner) contextForTask(t *task.Task) (c *taskctx.ExecutionContext, err error) {
	if t.Context == "" {
		return taskctx.DefaultContext(), nil
	}

	c, ok := r.contexts[t.Context]
	if !ok {
		return nil, fmt.Errorf("no such context %s", t.Context)
	}

	r.cleanupList.Store(t.Context, c)

	return c, nil
}

func (r *TaskRunner) checkTaskCondition(t *task.Task) (bool, error) {
	executionContext, err := r.contextForTask(t)
	if err != nil {
		return false, err
	}

	j, err := r.Compile(t.Condition, t, executionContext, nil, r.Stdout, r.Stderr, r.env, r.variables)
	if err != nil {
		return false, err
	}

	_, err = r.Executor.Execute(r.ctx, j)
	if err != nil {
		if executor.IsExitStatus(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *TaskRunner) storeTaskOutput(t *task.Task) {
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

func (r *TaskRunner) resolveDir(t *task.Task, executionCtx *taskctx.ExecutionContext) (string, error) {
	if t.Dir != "" {
		return utils.RenderString(t.Dir, r.variables.Merge(t.Variables).Map())
	} else if executionCtx.Dir != "" {
		return executionCtx.Dir, nil
	}

	return "", nil
}
