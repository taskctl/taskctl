package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/taskctl/taskctl/utils"
)

// Executor executes given job
type Executor interface {
	Execute(context.Context, *Job) ([]byte, error)
}

// DefaultExecutor is a default executor used for jobs
// Uses `mvdan.cc/sh/v3/interp` under the hood
type DefaultExecutor struct {
	dir    string
	env    []string
	interp *interp.Runner
	buf    bytes.Buffer
}

// NewDefaultExecutor creates new default executor
func NewDefaultExecutor(stdin io.Reader, stdout, stderr io.Writer) (*DefaultExecutor, error) {
	var err error
	e := &DefaultExecutor{
		env: os.Environ(),
	}

	e.dir, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	if stdout == nil {
		stdout = io.Discard
	}

	if stderr == nil {
		stderr = io.Discard
	}

	e.interp, err = interp.New(
		interp.StdIO(stdin, io.MultiWriter(&e.buf, stdout), io.MultiWriter(&e.buf, stderr)),
	)
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
	env = append(env, utils.ConvertEnv(utils.ConvertToMapOfStrings(job.Env.Map()))...)

	if job.Dir == "" {
		job.Dir = e.dir
	}

	slog.Debug(fmt.Sprintf("Executing \"%s\"", command))

	e.interp.Dir = job.Dir
	e.interp.Env = expand.ListEnviron(env...)

	var cancelFn context.CancelFunc
	if job.Timeout != nil {
		ctx, cancelFn = context.WithTimeout(ctx, *job.Timeout)
	}
	defer func() {
		if cancelFn != nil {
			cancelFn()
		}
	}()

	offset := e.buf.Len()
	err = e.interp.Run(ctx, cmd)
	if err != nil {
		return e.buf.Bytes()[offset:], err
	}

	return e.buf.Bytes()[offset:], nil
}

// IsExitStatus checks if given `err` is an exit status
func IsExitStatus(err error) (uint8, bool) {
	return interp.IsExitStatus(err)
}
