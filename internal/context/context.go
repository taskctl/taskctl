package context

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/taskctl/taskctl/internal/task"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/util"
)

type container struct {
	provider   string
	name       string
	image      string
	exec       bool
	options    []string
	env        []string
	executable util.Executable
}

type ssh struct {
	user       string
	host       string
	options    []string
	executable util.Executable
}

type ExecutionContext struct {
	ScheduledForCleanup bool

	ctxType    string
	executable util.Executable
	env        []string
	def        *config.ContextDefinition
	dir        string

	container container
	ssh       ssh

	up           []string
	down         []string
	before       []string
	after        []string
	startupError error

	onceUp   sync.Once
	onceDown sync.Once
	mu       sync.Mutex
}

func BuildContext(def *config.ContextDefinition, cfg *config.Config) (*ExecutionContext, error) {
	c := &ExecutionContext{
		ctxType: def.Type,
		executable: util.Executable{
			Bin:  def.Executable.Bin,
			Args: def.Executable.Args,
		},
		dir:    def.Dir,
		env:    append(os.Environ(), util.ConvertEnv(def.Env)...),
		def:    def,
		up:     def.Up,
		down:   def.Down,
		before: def.Before,
		after:  def.After,
	}

	switch c.ctxType {
	case config.ContextTypeContainer:
		buildContainerContext(def, cfg, c)
	case config.ContextTypeRemote:
		buildRemoteContext(def, cfg, c)
	}

	if c.dir == "" {
		var err error
		c.dir, err = os.Getwd()
		if err != nil {
			return c, nil
		}
	}

	if c.executable.Bin == "" {
		setDefaultShell(&c.executable)
	}

	return c, nil
}

func setDefaultShell(e *util.Executable) {
	e.Bin = "/bin/sh"
	e.Args = []string{"-c"}
}

func (c *ExecutionContext) Bin() string {
	return c.executable.Bin
}

func (c *ExecutionContext) Args() []string {
	return c.executable.Args
}

func (c *ExecutionContext) Env() []string {
	return c.env
}

func (c *ExecutionContext) WithEnvs(env []string) (*ExecutionContext, error) {
	def := *c.def
	for _, v := range env {
		kv := strings.Split(v, "=")
		if len(def.Env) == 0 {
			def.Env = make(map[string]string)
		}
		def.Env[kv[0]] = kv[1]
	}

	return BuildContext(&def, config.Get())
}

func (c *ExecutionContext) Up() error {
	c.onceUp.Do(func() {
		for _, command := range c.up {
			err := c.runServiceCommand(command)
			if err != nil {
				c.mu.Lock()
				c.startupError = err
				c.mu.Unlock()
				logrus.Errorf("context startup error: %s", err)
			}
		}
	})

	return c.startupError
}

func (c *ExecutionContext) Down() {
	c.onceDown.Do(func() {
		for _, command := range c.down {
			err := c.runServiceCommand(command)
			if err != nil {
				logrus.Errorf("context cleanup error: %s", err)
			}
		}
	})
}

func (c *ExecutionContext) Before() error {
	for _, command := range c.before {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) After() error {
	for _, command := range c.after {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) runServiceCommand(command string) (err error) {
	logrus.Debugf("running service context service command: %s", command)
	ca := strings.Split(command, " ")
	cmd := exec.Command(ca[0], ca[1:]...)
	cmd.Env = c.Env()
	cmd.Dir, err = os.Getwd()
	if err != nil {
		return err
	}

	out, err := cmd.Output()
	if err != nil {
		if exerr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("%v\n%s\n%s\n", err, out, exerr.Stderr)
		} else {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) BuildCommand(ctx context.Context, command string, t *task.Task) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch c.ctxType {
	case config.ContextTypeLocal:
		cmd = c.buildLocalCommand(ctx, command)
	case config.ContextTypeContainer:
		switch c.container.provider {
		case config.ContextContainerProviderDocker, config.ContextContainerProviderDockerCompose:
			cmd = c.buildDockerCommand(ctx, command)
		case config.ContextContainerProviderKubectl:
			cmd = c.buildKubectlCommand(ctx, command)
		}
	case config.ContextTypeRemote:
		cmd = c.buildRemoteCommand(ctx, command)
	default:
		return nil, errors.New("unknown context type")
	}

	if cmd == nil {
		return nil, errors.New("failed to build command")
	}

	if t != nil && t.Dir != "" {
		cmd.Dir = t.Dir
	}

	return cmd, nil
}

func (c *ExecutionContext) ScheduleForCleanup() {
	c.mu.Lock()
	c.ScheduledForCleanup = true
	c.mu.Unlock()
}
