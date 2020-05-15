package executor

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"mvdan.cc/sh/v3/expand"

	"github.com/sirupsen/logrus"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/taskctl/taskctl/internal/utils"
	"github.com/taskctl/taskctl/internal/variables"
)

type Executor interface {
	Execute(context.Context, *Job) ([]byte, error)
}

// linked list of jobs to Execute
type Job struct {
	Command string
	Dir     string
	Env     variables.Container
	Vars    variables.Container
	Timeout *time.Duration

	Stdout, Stderr io.Writer
	Stdin          io.Reader

	Next *Job
}

type DefaultExecutor struct {
	dir string
	env []string
}

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

func (e *DefaultExecutor) Execute(ctx context.Context, job *Job) ([]byte, error) {
	logrus.Debugf("Executing \"%s\"", job.Command)

	command, err := utils.RenderString(job.Command, job.Vars.Map())
	if err != nil {
		return nil, err
	}

	cmd, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, err
	}

	env := e.env
	env = append(env, utils.ConvertEnv(job.Env.Map())...)

	if job.Dir == "" {
		job.Dir = e.dir
	}

	buf := bytes.NewBuffer(make([]byte, 4096))
	r, err := interp.New(
		interp.Dir(job.Dir),
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(job.Stdin, io.MultiWriter(buf, job.Stdout), job.Stderr),
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
		logrus.Debug(err.Error())
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

func IsExitStatus(err error) (uint8, bool) {
	return interp.IsExitStatus(err)
}
