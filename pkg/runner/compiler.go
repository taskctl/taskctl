package runner

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/executor"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/sirupsen/logrus"
)

// TaskCompiler compiles tasks into jobs for executor
type TaskCompiler struct {
	variables variables.Container
}

// NewTaskCompiler create new TaskCompiler instance
func NewTaskCompiler() *TaskCompiler {
	return &TaskCompiler{variables: variables.NewVariables()}
}

// CompileTask compiles task into Job (linked list of commands) executed by Executor
func (tc *TaskCompiler) CompileTask(t *task.Task, executionContext *ExecutionContext, stdin io.Reader, stdout, stderr io.Writer, env, vars variables.Container) (*executor.Job, error) {
	vars = t.Variables.Merge(vars)

	var job, prev *executor.Job

	for k, v := range vars.Map() {
		if reflect.ValueOf(v).Kind() != reflect.String {
			continue
		}

		v, err := utils.RenderString(v.(string), vars.Map())
		if err != nil {
			return nil, err
		}
		vars.Set(k, v)
	}

	// creating multiple versions of the same task with different env input
	for _, variant := range t.GetVariations() {
		// each command in the array needs compiling
		for _, command := range t.Commands {
			j, err := tc.CompileCommand(
				t.Name,
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

// CompileCommand compiles command into Job
func (tc *TaskCompiler) CompileCommand(
	taskName string,
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

	commandArgs := []string{}

	if executionCtx.Executable != nil {
		commandArgs = executionCtx.Executable.GetArgs()
	}
	// Look at the executable details and check if the command is running `docker` determine if an Envfile is being generated
	// If it has then check to see if the args contains the --env-file flag and if does modify the path to the envfile
	// if it does not then add the --env-file flag to the args array
	if executionCtx.Envfile != nil && executionCtx.Envfile.Generate {

		// define the filename to hold the envfile path
		// get the timestamp to use to append to the envfile name
		filename := utils.GetFullPath(
			filepath.Join(
				executionCtx.Envfile.GeneratedDir,
				fmt.Sprintf("generated_%s_%v.env", utils.ConvertStringToMachineFriendly(taskName), time.Now().UnixNano()),
			))

		commandArgs = executionCtx.Executable.BuildArgsWithEnvFile(filename)
		// set the path to the generated envfile
		executionCtx.Envfile.Path = filename
		// generate the envfile with supplied env only
		err := executionCtx.GenerateEnvfile(env)
		if err != nil {
			return nil, err
		}
	}

	c := []string{command}
	if executionCtx.Executable != nil {
		c = []string{executionCtx.Executable.Bin}
		c = append(c, commandArgs...)
		c = append(c, fmt.Sprintf("%s%s%s", executionCtx.Quote, command, executionCtx.Quote))
	}

	j.Command = strings.Join(c, " ")

	logrus.Debugf("command: %s", j.Command)

	var err error
	if dir != "" {
		j.Dir = dir
	} else if executionCtx.Dir != "" {
		j.Dir = executionCtx.Dir
	}

	j.Dir, err = utils.RenderString(j.Dir, j.Vars.Map())
	if err != nil {
		return nil, err
	}

	return j, nil
}
