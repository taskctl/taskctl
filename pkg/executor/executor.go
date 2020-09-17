package executor

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"

	"github.com/sirupsen/logrus"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/taskctl/taskctl/pkg/utils"
)

// Executor executes given job
type Executor interface {
	Execute(context.Context, *Job) ([]byte, error)
}

// DefaultExecutor is a default executor used for jobs
// Uses `mvdan.cc/sh/v3/interp` under the hood
type DefaultExecutor struct {
	dir string
	env []string
}

// NewDefaultExecutor creates new default executor
func NewDefaultExecutor() (*DefaultExecutor, error) {
	var err error
	e := &DefaultExecutor{
		env: os.Environ(),
	}

	e.dir, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	return e, nil
}

// Execute executes given job with provided context
// Returns job output
func (e *DefaultExecutor) Execute(ctx context.Context, job *Job) ([]byte, error) {
	command, err := utils.RenderString(job.Command, job.Vars.Map())
	if err != nil {
		return nil, err
	}

	cmd, err := syntax.NewParser(syntax.KeepComments(true)).Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, err
	}

	env := e.env
	env = append(env, utils.ConvertEnv(job.Env.Map())...)

	if job.Dir == "" {
		job.Dir = e.dir
	}

	logrus.Debugf("Executing \"%s\"", command)

	stdout := job.Stdout
	if stdout == nil {
		stdout = ioutil.Discard
	}

	stderr := job.Stderr
	if stderr == nil {
		stderr = ioutil.Discard
	}

	var buf bytes.Buffer
	r, err := interp.New(
		interp.Dir(job.Dir),
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(job.Stdin, io.MultiWriter(&buf, stdout), io.MultiWriter(&buf, stderr)),
	)
	if err != nil {
		return nil, err
	}

	var cancelFn context.CancelFunc
	if job.Timeout != nil {
		ctx, cancelFn = context.WithTimeout(ctx, *job.Timeout)
	}
	defer func() {
		if cancelFn != nil {
			cancelFn()
		}
	}()

	err = r.Run(ctx, cmd)
	if err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

// IsExitStatus checks if given `err` is an exit status
func IsExitStatus(err error) (uint8, bool) {
	return interp.IsExitStatus(err)
}
