// Package executor runs compiled jobs through an embedded shell interpreter, without a system shell.
package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"strings"

	"mvdan.cc/sh/v3/expand"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/taskctl/taskctl/internal/envutil"
	"github.com/taskctl/taskctl/internal/tmpl"
)

// Executor executes given job
type Executor interface {
	Execute(context.Context, *Job) ([]byte, error)
}

// DefaultExecutor is a default executor used for jobs
// Uses `mvdan.cc/sh/v3/interp` under the hood
type DefaultExecutor struct {
	// DryRun makes Execute render and parse the command to validate it, then
	// return without executing it.
	DryRun bool

	dir     string
	env     []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	buf     bytes.Buffer
	interp  *interp.Runner
	lastEnv map[string]string
	lastDir string
}

// NewDefaultExecutor creates new default executor
func NewDefaultExecutor(stdin io.Reader, stdout, stderr io.Writer) (*DefaultExecutor, error) {
	var err error
	e := &DefaultExecutor{
		env: envutil.SanitizeEnviron(os.Environ()),
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

	e.stdin = stdin
	e.stdout = io.MultiWriter(&e.buf, stdout)
	e.stderr = io.MultiWriter(&e.buf, stderr)

	return e, nil
}

// Execute executes given job with provided context
// Returns job output
func (e *DefaultExecutor) Execute(ctx context.Context, job *Job) ([]byte, error) {
	command, err := tmpl.RenderString(job.Command, job.Vars.Map())
	if err != nil {
		return nil, err
	}

	cmd, err := syntax.NewParser(syntax.KeepComments(true)).Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, err
	}

	// Dry run validates the command (render + parse above) but skips execution.
	if e.DryRun {
		return nil, nil
	}

	jobEnv := envutil.ConvertToMapOfStrings(job.Env.Map())

	if job.Dir == "" {
		job.Dir = e.dir
	}

	slog.Debug(fmt.Sprintf("Executing \"%s\"", command))

	// Reuse the interpreter while the environment and directory are unchanged so
	// shell state (functions, variables, cwd) carries across a task's commands;
	// rebuild it when either changes (a new variation) so each variation runs
	// with its own environment/directory and a clean state.
	if e.interp == nil || job.Dir != e.lastDir || !maps.Equal(jobEnv, e.lastEnv) {
		env := envutil.OverlayEnviron(e.env, jobEnv)
		e.interp, err = interp.New(
			interp.StdIO(e.stdin, e.stdout, e.stderr),
			interp.Dir(job.Dir),
			interp.Env(expand.ListEnviron(env...)),
		)
		if err != nil {
			return nil, err
		}
		e.lastEnv = jobEnv
		e.lastDir = job.Dir
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

	offset := e.buf.Len()
	err = e.interp.Run(ctx, cmd)
	if err != nil {
		return e.buf.Bytes()[offset:], err
	}

	return e.buf.Bytes()[offset:], nil
}

// IsExitStatus checks if given `err` is an exit status
func IsExitStatus(err error) (uint8, bool) {
	var status interp.ExitStatus
	if errors.As(err, &status) {
		return uint8(status), true
	}
	return 0, false
}
