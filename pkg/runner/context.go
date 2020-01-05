package runner

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/internal/config"
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
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
	ctxType    string
	executable util.Executable
	env        []string
	def        *builder.ContextDefinition
	dir        string

	container container
	ssh       ssh

	up     []string
	down   []string
	before []string
	after  []string

	scheduledForCleanup bool

	onceUp   sync.Once
	onceDown sync.Once
	mu       sync.Mutex
}

func BuildContext(def *builder.ContextDefinition, wcfg *builder.WilsonConfigDefinition) (*ExecutionContext, error) {
	c := &ExecutionContext{
		ctxType: def.Type,
		executable: util.Executable{
			Bin:  def.Bin,
			Args: def.Args,
		},
		container: container{
			provider: def.Container.Provider,
			name:     def.Container.Name,
			image:    def.Container.Image,
			exec:     def.Container.Exec,
			options:  def.Container.Options,
			env:      util.ConvertEnv(def.Container.Env),
			executable: util.Executable{
				Bin:  def.Container.Bin,
				Args: def.Container.Args,
			},
		},
		ssh: ssh{
			user:    def.SSH.User,
			host:    def.SSH.Host,
			options: def.SSH.Options,
			executable: util.Executable{
				Bin:  def.SSH.Bin,
				Args: def.SSH.Options,
			},
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
		switch c.container.provider {
		case config.ContextContainerProviderDocker:
			if c.container.executable.Bin == "" {
				if wcfg.Docker.Bin != "" {
					c.container.executable.Bin = wcfg.Docker.Bin
				} else {
					c.container.executable.Bin = "docker"
				}
			}
			if len(c.container.executable.Args) == 0 {
				c.container.executable.Args = wcfg.Docker.Args
			}
		case config.ContextContainerProviderDockerCompose:
			if c.container.executable.Bin == "" {
				if wcfg.DockerCompose.Bin != "" {
					c.container.executable.Bin = wcfg.DockerCompose.Bin
				} else {
					c.container.executable.Bin = "docker-compose"
				}
			}
			if len(c.container.executable.Args) == 0 {
				c.container.executable.Args = wcfg.DockerCompose.Args
			}
		case config.ContextContainerProviderKubectl:
			if c.container.executable.Bin == "" {
				if wcfg.Kubectl.Bin != "" {
					c.container.executable.Bin = wcfg.Kubectl.Bin
				} else {
					c.container.executable.Bin = "kubectl"
				}
			}

			if len(c.container.executable.Args) == 0 {
				c.container.executable.Args = wcfg.Kubectl.Args
			}
		}
	case config.ContextTypeRemote:
		if c.ssh.executable.Bin == "" {
			if wcfg.Ssh.Bin != "" {
				c.ssh.executable.Bin = wcfg.Ssh.Bin
			} else {
				c.ssh.executable.Bin = "ssh"
			}
		}

		if len(c.ssh.executable.Args) == 0 {
			c.ssh.executable.Args = wcfg.Ssh.Args
		}

		c.ssh.executable.Args = append(c.ssh.executable.Args, "-T")

		if c.ssh.user != "" {
			c.ssh.executable.Args = append(c.ssh.executable.Args, fmt.Sprintf("%s@%s", c.ssh.user, c.ssh.host))
		} else {
			c.ssh.executable.Args = append(c.ssh.executable.Args, c.ssh.host)
		}
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

func (c *ExecutionContext) buildLocalCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.executable.Bin, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)
	cmd.Env = c.env
	cmd.Dir = c.dir

	return cmd
}

func (c *ExecutionContext) buildDockerCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.container.executable.Bin, c.container.executable.Args...)
	cmd.Env = c.env
	cmd.Dir = c.dir

	switch c.container.provider {
	case config.ContextContainerProviderDocker:
		if c.container.exec {
			cmd.Args = append(cmd.Args, "exec")
			for _, v := range c.container.env {
				cmd.Args = append(cmd.Args, "-e", v)
			}
			cmd.Args = append(cmd.Args, c.container.options...)
			cmd.Args = append(cmd.Args, c.container.name)
		} else {
			cmd.Args = append(cmd.Args, "run", "--rm")
			if c.container.name != "" {
				cmd.Args = append(cmd.Args, "--name", c.container.name)
			}
			for _, v := range c.container.env {
				cmd.Args = append(cmd.Args, "-e", v)
			}
			cmd.Args = append(cmd.Args, c.container.options...)
			cmd.Args = append(cmd.Args, c.container.image)
		}
	case config.ContextContainerProviderDockerCompose:
		if c.container.exec {
			cmd.Args = append(cmd.Args, "exec", "-T")
		} else {
			cmd.Args = append(cmd.Args, "run", "--rm")
		}

		cmd.Args = append(cmd.Args, c.container.options...)
		for _, v := range c.container.env {
			cmd.Args = append(cmd.Args, "-e", v)
		}

		cmd.Args = append(cmd.Args, c.container.name)
	}

	cmd.Args = append(cmd.Args, c.executable.Bin)
	cmd.Args = append(cmd.Args, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)

	return cmd
}

func (c *ExecutionContext) buildKubectlCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.container.executable.Bin, c.container.executable.Args...)
	cmd.Env = append(c.env, c.container.env...)
	cmd.Dir = c.dir

	cmd.Args = append(cmd.Args, "exec", c.container.name)
	cmd.Args = append(cmd.Args, c.container.options...)
	cmd.Args = append(cmd.Args, "--")
	cmd.Args = append(cmd.Args, c.executable.Bin)
	cmd.Args = append(cmd.Args, c.executable.Args...)
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s %s", strings.Join(c.container.env, " "), command))

	return cmd
}

func (c *ExecutionContext) buildRemoteCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.ssh.executable.Bin, c.ssh.executable.Args...)
	cmd.Env = c.env
	cmd.Dir = c.dir

	cmd.Args = append(cmd.Args, c.executable.Bin)
	cmd.Args = append(cmd.Args, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)

	return cmd
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

	return BuildContext(&def, &config.Get().WilsonConfigDefinition)
}

func (c *ExecutionContext) Up() {
	c.onceUp.Do(func() {
		for _, command := range c.up {
			err := c.runServiceCommand(command)
			if err != nil {
				log.Errorf("context startup error: %s", err)
			}
		}
	})
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
	cmd.Dir, err = util.Getcwd()
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

func (c *ExecutionContext) createCommand(ctx context.Context, command string) (*exec.Cmd, error) {
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
	c.scheduledForCleanup = true
	c.mu.Unlock()
}
