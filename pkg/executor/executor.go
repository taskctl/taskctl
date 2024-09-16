// package executor
//
// It uses the mvdan.sh shell implementation in Go.
// injects a custom environment per execution
//
// not all *nix* commands are available, should only be used for a limited number of scenarios
package executor

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"

	"github.com/sirupsen/logrus"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/utils"
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
	outBuf *bytes.Buffer
	errBuf *bytes.Buffer
	// doReset resets the execution environment after each run
	doReset bool
}

// NewDefaultExecutor creates new default executor
func NewDefaultExecutor(stdin io.Reader, stdout, stderr io.Writer) (*DefaultExecutor, error) {
	var err error
	e := &DefaultExecutor{
		env: os.Environ(), // do not want to set the environment here
	}

	e.dir, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	e.outBuf = &bytes.Buffer{}
	e.errBuf = &bytes.Buffer{}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if _, ok := stdout.(*output.SafeWriter); !ok {
		stdout = output.NewSafeWriter(stdout)
	}

	if _, ok := stderr.(*output.SafeWriter); !ok {
		stderr = output.NewSafeWriter(stderr)
	}

	e.interp, err = interp.New(
		interp.StdIO(stdin, io.MultiWriter(output.NewSafeWriter(e.outBuf), stdout), io.MultiWriter(output.NewSafeWriter(e.errBuf), stderr)),
	)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *DefaultExecutor) WithReset(doReset bool) *DefaultExecutor {
	e.doReset = doReset
	return e
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

	logrus.Debugf("Executing \"%s\"", command)

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

	// TODO: come back to this
	// offset := e.buf.Len()

	// Reset needs to be called before Run
	// even the first time around else the vars won't be cleared correctly
	// and re-injected by the mvdan shell
	if e.doReset {
		e.interp.Reset()
	}

	if err := e.interp.Run(ctx, cmd); err != nil {
		return append(e.outBuf.Bytes(), e.errBuf.Bytes()...), err
	}
	return append(e.outBuf.Bytes(), e.errBuf.Bytes()...), nil
}

// IsExitStatus checks if given `err` is an exit status
func IsExitStatus(err error) (uint8, bool) {
	return interp.IsExitStatus(err)
}
