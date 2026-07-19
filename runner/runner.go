// Package runner compiles tasks into jobs and executes them inside execution contexts.
package runner

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"golang.org/x/text/language"

	"golang.org/x/text/cases"

	"github.com/taskctl/taskctl/executor"
	"github.com/taskctl/taskctl/internal/collections"

	"github.com/taskctl/taskctl/variables"

	"github.com/taskctl/taskctl/internal/output"

	"github.com/taskctl/taskctl/task"
)

// defaultContextName is the context a task falls back to when it declares no
// context: if the config defines a context by this name it is used (so its env,
// variables and hooks apply to every such task), otherwise an empty context is.
const defaultContextName = "default"

// Runner describes tasks runner interface
type Runner interface {
	Run(t *task.Task) error
	Cancel()
	Finish()
}

// TaskRunner run tasks
type TaskRunner struct {
	// DryRun makes each task's commands (condition, before, main, after) render
	// and parse for validation but not execute, so a task with valid commands is
	// marked completed (an invalid template or command still fails). Context
	// lifecycle hooks (Up/Down/Before/After) are not skipped.
	DryRun    bool
	contexts  map[string]*ExecutionContext
	variables variables.Container
	env       variables.Container

	ctx         context.Context
	cancelFunc  context.CancelFunc
	cancelMutex sync.RWMutex
	canceling   bool
	doneCh      chan struct{}

	results collections.SyncMap[string, taskResult]

	compiler *taskCompiler

	Stdin          io.Reader
	Stdout, Stderr io.Writer
	OutputFormat   string

	cleanupList collections.SyncMap[string, *ExecutionContext]
}

// taskInfo and contextInfo are the template-facing views of the running task and
// its context (.Task and .Context). The types are unexported but their fields
// are exported: text/template navigates the fields, not the type name. Only
// static task metadata is exposed — not runtime state (exit code, logs), the
// variable/env containers, or the raw (unrendered) command slices.
type taskInfo struct {
	Name         string
	Description  string
	Dir          string
	Context      string
	Condition    string
	Timeout      *time.Duration
	AllowFailure bool
	Interactive  bool
	ExportAs     string
}

type contextInfo struct {
	Name       string
	Dir        string
	Executable *Binary
}

// taskResult is the template-facing view of a completed task's result
// (.Tasks.<Name>). Exported fields so text/template can read them.
type taskResult struct {
	Stdout   string
	Stderr   string
	ExitCode int16
}

// NewTaskRunner creates new TaskRunner instance
func NewTaskRunner(opts ...Opts) (*TaskRunner, error) {
	r := &TaskRunner{
		compiler:     newTaskCompiler(),
		OutputFormat: output.FormatRaw,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		variables:    variables.NewVariables(),
		env:          variables.NewVariables(),
		doneCh:       make(chan struct{}, 1),
	}

	r.ctx, r.cancelFunc = context.WithCancel(context.Background())

	for _, o := range opts {
		o(r)
	}

	r.env = variables.FromMap(map[string]string{"ARGS": r.variables.Get("Args").(string)})

	return r, nil
}

// Run run provided task.
// TaskRunner first compiles task into linked list of Jobs, then passes those jobs to Executor.
// Any failure — including errors before execution starts (context resolution,
// hooks, compilation) — is recorded on the task via Errored/Error.
func (r *TaskRunner) Run(t *task.Task) (err error) {
	defer func() {
		// Pre-execution failures return an error without reaching execute(),
		// which is what normally marks the task; record them here so task
		// status and error reporting reflect the failure.
		if err != nil && !t.Errored {
			t.Errored = true
			t.Error = err
		}

		r.cancelMutex.RLock()
		if r.canceling {
			close(r.doneCh)
		}
		r.cancelMutex.RUnlock()
	}()

	if err := r.ctx.Err(); err != nil {
		return err
	}

	execContext, err := r.contextForTask(r.ctx, t)
	if err != nil {
		return err
	}

	outputFormat := r.OutputFormat

	var stdin io.Reader
	if t.Interactive {
		outputFormat = output.FormatRaw
		stdin = r.Stdin
	}

	taskOutput, err := output.NewTaskOutput(t, outputFormat, r.Stdout, r.Stderr)
	if err != nil {
		return err
	}

	defer func() {
		err := taskOutput.Finish()
		if err != nil {
			slog.Error(err.Error())
		}

		err = execContext.After()
		if err != nil {
			slog.Error(err.Error())
		}
	}()

	vars := r.variables.Merge(execContext.Variables).Merge(t.Variables)
	vars.Set("Task", taskInfo{
		Name:         t.Name,
		Description:  t.Description,
		Dir:          t.Dir,
		Context:      t.Context,
		Condition:    t.Condition,
		Timeout:      t.Timeout,
		AllowFailure: t.AllowFailure,
		Interactive:  t.Interactive,
		ExportAs:     t.ExportAs,
	})
	vars.Set("Context", contextInfo{Name: t.Context, Dir: execContext.Dir, Executable: execContext.Executable})
	vars.Set("Tasks", r.results.Snapshot())

	env := r.env.Merge(execContext.Env)
	env = env.With("TASK_NAME", t.Name)
	env = env.Merge(t.Env)

	meets, err := r.checkTaskCondition(t, env, vars)
	if err != nil {
		return err
	}

	if !meets {
		slog.Info(fmt.Sprintf("task %s was skipped", t.Name))
		t.Skipped = true
		return nil
	}

	err = r.before(r.ctx, t, env, vars)
	if err != nil {
		return err
	}

	job, err := r.compiler.compileTask(t, execContext, stdin, taskOutput.Stdout(), taskOutput.Stderr(), env, vars)
	if err != nil {
		return err
	}

	err = taskOutput.Start()
	if err != nil {
		return err
	}

	err = r.execute(r.ctx, t, job)

	// execute leaves a succeeded task's exit code at -1; normalize it before the
	// result is stored and the footer is written. Failures keep their real code.
	if !t.Errored && t.ExitCode < 0 {
		t.ExitCode = 0
	}
	r.storeTaskResult(t)

	if err != nil {
		return err
	}

	return r.after(r.ctx, t, env, vars)
}

// Cancel cancels execution
func (r *TaskRunner) Cancel() {
	r.cancelMutex.Lock()
	if !r.canceling {
		r.canceling = true
		defer slog.Debug("runner has been cancelled")
		r.cancelFunc()
	}
	r.cancelMutex.Unlock()
	<-r.doneCh
}

// Finish makes cleanup tasks over contexts
func (r *TaskRunner) Finish() {
	r.cleanupList.Range(func(key string, value *ExecutionContext) bool {
		value.Down()
		return true
	})
	output.Close()
}

func (r *TaskRunner) before(ctx context.Context, t *task.Task, env, vars variables.Container) error {
	if len(t.Before) == 0 {
		return nil
	}

	execContext, err := r.contextForTask(ctx, t)
	if err != nil {
		return err
	}

	for _, command := range t.Before {
		job, err := r.compiler.compileCommand(command, execContext, t.Dir, t.Timeout, nil, r.Stdout, r.Stderr, env, vars)
		if err != nil {
			return fmt.Errorf("\"before\" command compilation failed: %w", err)
		}

		exec, err := executor.NewDefaultExecutor(job.Stdin, job.Stdout, job.Stderr)
		if err != nil {
			return err
		}
		exec.DryRun = r.DryRun

		_, err = exec.Execute(ctx, job)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TaskRunner) after(ctx context.Context, t *task.Task, env, vars variables.Container) error {
	if len(t.After) == 0 {
		return nil
	}

	execContext, err := r.contextForTask(ctx, t)
	if err != nil {
		return err
	}

	for _, command := range t.After {
		job, err := r.compiler.compileCommand(command, execContext, t.Dir, t.Timeout, nil, r.Stdout, r.Stderr, env, vars)
		if err != nil {
			return fmt.Errorf("\"after\" command compilation failed: %w", err)
		}

		exec, err := executor.NewDefaultExecutor(job.Stdin, job.Stdout, job.Stderr)
		if err != nil {
			return err
		}
		exec.DryRun = r.DryRun

		_, err = exec.Execute(ctx, job)
		if err != nil {
			slog.Warn(err.Error())
		}
	}

	return nil
}

func (r *TaskRunner) contextForTask(ctx context.Context, t *task.Task) (c *ExecutionContext, err error) {
	name := t.Context
	if name == "" {
		name = defaultContextName
	}

	c, ok := r.contexts[name]
	switch {
	case ok:
		r.cleanupList.Store(name, c)
	case t.Context != "":
		return nil, fmt.Errorf("no such context %s", t.Context)
	default:
		c = defaultContext()
	}

	err = c.Up(ctx)
	if err != nil {
		return nil, err
	}

	err = c.Before(ctx)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (r *TaskRunner) checkTaskCondition(t *task.Task, env, vars variables.Container) (bool, error) {
	if t.Condition == "" {
		return true, nil
	}

	executionContext, err := r.contextForTask(r.ctx, t)
	if err != nil {
		return false, err
	}

	job, err := r.compiler.compileCommand(t.Condition, executionContext, t.Dir, t.Timeout, nil, r.Stdout, r.Stderr, env, vars)
	if err != nil {
		return false, err
	}

	exec, err := executor.NewDefaultExecutor(job.Stdin, job.Stdout, job.Stderr)
	if err != nil {
		return false, err
	}
	exec.DryRun = r.DryRun

	_, err = exec.Execute(context.Background(), job)
	if err != nil {
		if _, ok := executor.IsExitStatus(err); ok {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (r *TaskRunner) storeTaskResult(t *task.Task) {
	stdout := t.Stdout()
	stderr := t.Stderr()

	if t.ExportAs != "" {
		r.env.Set(t.ExportAs, stdout)
	}

	r.results.Store(cases.Title(language.English).String(t.Name), taskResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: t.ExitCode,
	})
}

func (r *TaskRunner) execute(ctx context.Context, t *task.Task, job *executor.Job) error {
	exec, err := executor.NewDefaultExecutor(job.Stdin, job.Stdout, job.Stderr)
	if err != nil {
		return err
	}
	exec.DryRun = r.DryRun

	t.Start = time.Now()
	var prevOutput []byte
	for nextJob := job; nextJob != nil; nextJob = nextJob.Next {
		var err error
		nextJob.Vars.Set("Output", string(prevOutput))

		prevOutput, err = exec.Execute(ctx, nextJob)
		if err != nil {
			slog.Debug(err.Error())
			if status, ok := executor.IsExitStatus(err); ok {
				t.ExitCode = int16(status)
				if t.AllowFailure {
					continue
				}
			}
			t.Errored = true
			t.Error = err
			t.End = time.Now()
			return t.Error
		}
	}
	t.End = time.Now()

	return nil
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
		runner.compiler.variables = variables
	}
}
