package context

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/pkg/builder"
	"github.com/taskctl/taskctl/pkg/util"
	"os"
	"os/exec"
	"strings"
	"sync"
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
	def        *builder.ContextDefinition
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

func BuildContext(def *builder.ContextDefinition, wcfg *builder.TaskctlConfigDefinition) (*ExecutionContext, error) {
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
		buildContainerContext(def, wcfg, c)
	case config.ContextTypeRemote:
		buildRemoteContext(def, wcfg, c)
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

	return BuildContext(&def, &config.Get().TaskctlConfigDefinition)
}

func (c *ExecutionContext) Up() error {
	c.onceUp.Do(func() {
		for _, command := range c.up {
			err := c.runServiceCommand(command)
			if err != nil {
				c.mu.Lock()
				c.startupError = err
				c.mu.Unlock()
				log.Errorf("context startup error: %s", err)
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
				log.Errorf("context cleanup error: %s", err)
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
	log.Debugf("running service context service command: %s", command)
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

func (c *ExecutionContext) CreateCommand(ctx context.Context, command string) (*exec.Cmd, error) {
	switch c.ctxType {
	case config.ContextTypeLocal:
		return c.buildLocalCommand(ctx, command), nil
	case config.ContextTypeContainer:
		switch c.container.provider {
		case config.ContextContainerProviderDocker, config.ContextContainerProviderDockerCompose:
			return c.buildDockerCommand(ctx, command), nil
		case config.ContextContainerProviderKubectl:
			return c.buildKubectlCommand(ctx, command), nil
		}
	case config.ContextTypeRemote:
		return c.buildRemoteCommand(ctx, command), nil
	default:
		return nil, errors.New("unknown context type")
	}

	return nil, nil
}

func (c *ExecutionContext) ScheduleForCleanup() {
	c.mu.Lock()
	c.ScheduledForCleanup = true
	c.mu.Unlock()
}
