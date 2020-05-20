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

	"github.com/taskctl/taskctl/pkg/executor"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/output"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/task"
	"github.com/taskctl/taskctl/pkg/utils"
)

// Runner describes tasks runner interface
type Runner interface {
	Run(t *task.Task) error
	Cancel()
	Finish()
}

// TaskRunner run tasks
type TaskRunner struct {
	Executor  executor.Executor
	DryRun    bool
	contexts  map[string]*ExecutionContext
	variables variables.Container
	env       variables.Container

	ctx        context.Context
	cancelFunc context.CancelFunc

	Stdin          io.Reader
	Stdout, Stderr io.Writer
	OutputFormat   string

	cleanupList sync.Map
}

// NewTaskRunner creates new TaskRunner instance
func NewTaskRunner(opts ...Opts) (*TaskRunner, error) {
	exec, err := executor.NewDefaultExecutor()
	if err != nil {
		return nil, err
	}

	r := &TaskRunner{
		Executor:     exec,
		OutputFormat: output.FormatRaw,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		variables:    variables.NewVariables(),
		env:          variables.NewVariables(),
	}

	r.ctx, r.cancelFunc = context.WithCancel(context.Background())

	for _, o := range opts {
		o(r)
	}

	r.env = variables.FromMap(map[string]string{"ARGS": r.variables.Get("Args")})

	return r, nil
}

// SetContexts sets task runner's contexts
func (r *TaskRunner) SetContexts(contexts map[string]*ExecutionContext) *TaskRunner {
	r.contexts = contexts

	return r
}

// SetVariables sets task runner's variables
func (r *TaskRunner) SetVariables(contexts map[string]*ExecutionContext) *TaskRunner {
	r.contexts = contexts

	return r
}

// Run run provided task.
// TaskRunner first compiles task into linked list of Jobs, then passes those jobs to Executor
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

		if t.ExitCode == -1 && !t.Errored {
			t.ExitCode = 0
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
		outputFormat = output.FormatRaw
	}

	taskOutput, err := output.NewTaskOutput(t, outputFormat, r.Stdout, r.Stderr)
	if err != nil {
		return err
	}

	defer func() {
		err := taskOutput.Finish()
		if err != nil {
			logrus.Warning(err)
		}
	}()

	var stdin io.Reader
	if t.Interactive {
		stdin = r.Stdin
	}

	var job, prev *executor.Job
	for _, variant := range t.GetVariations() {
		for _, command := range t.Commands {
			j, err := r.Compile(
				command,
				t,
				execContext,
				stdin,
				taskOutput.Stdout(),
				taskOutput.Stderr(),
				env.Merge(variables.FromMap(variant)),
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
			if status, ok := executor.IsExitStatus(err); ok {
				t.ExitCode = int16(status)
				if t.AllowFailure {
					continue
				}
			}
			t.Errored = true
			t.Error = err
			break
		}
	}
	t.End = time.Now()

	if t.Errored {
		return t.Error
	}

	r.storeTaskOutput(t)

	if len(t.After) > 0 {
		err = r.after(t, env, vars)
		if err != nil {
			return err
		}
	}

	return nil
}

// Cancel cancels execution
func (r *TaskRunner) Cancel() {
	logrus.Debug("Runner has been cancelled")
	r.cancelFunc()
}

// Finish makes cleanup tasks over contexts
func (r *TaskRunner) Finish() {
	r.cleanupList.Range(func(key, value interface{}) bool {
		value.(*ExecutionContext).Down()
		return true
	})
	output.Close()
}

// Compile compiles task into Job executed by Executor
func (r *TaskRunner) Compile(command string, t *task.Task, executionCtx *ExecutionContext, stdin io.Reader, stdout, stderr io.Writer, env, vars variables.Container) (*executor.Job, error) {
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

// WithVariable adds variable to task runner's variables list.
// It creates new instance of variables container.
func (r *TaskRunner) WithVariable(key, value string) *TaskRunner {
	r.variables = r.variables.With(key, value)

	return r
}

func (r *TaskRunner) after(t *task.Task, env, vars variables.Container) error {
	execContext, err := r.contextForTask(t)
	if err != nil {
		return err
	}

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

	return nil
}

func (r *TaskRunner) contextForTask(t *task.Task) (c *ExecutionContext, err error) {
	if t.Context == "" {
		return DefaultContext(), nil
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
		if _, ok := executor.IsExitStatus(err); ok {
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

func (r *TaskRunner) resolveDir(t *task.Task, executionCtx *ExecutionContext) (string, error) {
	if t.Dir != "" {
		return utils.RenderString(t.Dir, r.variables.Merge(t.Variables).Map())
	} else if executionCtx.Dir != "" {
		return executionCtx.Dir, nil
	}

	return "", nil
}

// Opts is a task runner configuration function.
type Opts func(*TaskRunner)

// WithContexts adds provided contexts to task runner
func WithContexts(contexts map[string]*ExecutionContext) Opts {
	return func(runner *TaskRunner) {
		runner.contexts = contexts
	}
}

// WithVariables adds provided variables to task runner
func WithVariables(variables variables.Container) Opts {
	return func(runner *TaskRunner) {
		runner.variables = variables
	}
}
