package runner

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/taskctl/taskctl/executor"
	"github.com/taskctl/taskctl/internal/envutil"
	"github.com/taskctl/taskctl/internal/tmpl"
	"github.com/taskctl/taskctl/task"
	"github.com/taskctl/taskctl/variables"
)

// taskCompiler compiles tasks into jobs for executor
type taskCompiler struct {
	variables variables.Container
}

func newTaskCompiler() *taskCompiler {
	return &taskCompiler{variables: variables.NewVariables()}
}

// compileTask compiles task into Job (linked list of commands) executed by Executor
func (tc *taskCompiler) compileTask(t *task.Task, executionContext *ExecutionContext, stdin io.Reader, stdout, stderr io.Writer, env, vars variables.Container) (*executor.Job, error) {
	vars = t.Variables.Merge(vars)
	var job, prev *executor.Job

	for k, v := range vars.Map() {
		if reflect.ValueOf(v).Kind() != reflect.String {
			continue
		}

		v, err := tmpl.RenderString(v.(string), vars.Map())
		if err != nil {
			return nil, err
		}
		vars.Set(k, v)
	}

	for _, variant := range t.GetVariations() {
		for _, command := range t.Commands {
			j, err := tc.compileCommand(
				command,
				executionContext,
				t.Dir,
				t.Timeout,
				stdin,
				stdout,
				stderr,
				env.Merge(variables.FromMap(variant)),
				vars,
			)
			if err != nil {
				return nil, err
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

	return job, nil
}

func (tc *taskCompiler) compileCommand(
	command string,
	executionCtx *ExecutionContext,
	dir string,
	timeout *time.Duration,
	stdin io.Reader,
	stdout, stderr io.Writer,
	env, vars variables.Container,
) (*executor.Job, error) {
	j := &executor.Job{
		Timeout: timeout,
		Env:     env,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
		Vars:    tc.variables.Merge(vars),
	}

	var renderedDir string
	if dir != "" {
		renderedDir = dir
	} else if executionCtx.Dir != "" {
		renderedDir = executionCtx.Dir
	}

	renderedDir, err := tmpl.RenderString(renderedDir, j.Vars.Map())
	if err != nil {
		return nil, err
	}

	if executionCtx.wrapper != nil {
		j.Command = executionCtx.wrapper.wrap(command, envutil.ConvertToMapOfStrings(env.Map()), renderedDir)
		j.Env = variables.NewVariables()
		j.Dir = ""
	} else {
		var c []string
		if executionCtx.Executable != nil {
			c = []string{executionCtx.Executable.Bin}
			c = append(c, executionCtx.Executable.Args...)
			c = append(c, fmt.Sprintf("%s%s%s", executionCtx.Quote, command, executionCtx.Quote))
		} else {
			c = []string{command}
		}

		j.Command = strings.Join(c, " ")
		j.Dir = renderedDir
	}

	return j, nil
}
