package runner

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/config"
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

type Context struct {
	ctxType    string
	executable util.Executable
	env        []string
	def        *config.ContextConfig
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

func BuildContext(def config.ContextConfig, wcfg *config.WilsonConfig) (*Context, error) {
	c := &Context{
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
			executable: struct {
				Bin  string
				Args []string
			}{Bin: def.Container.Bin, Args: def.Container.Args},
		},
		ssh: ssh{
			user:    def.Ssh.User,
			host:    def.Ssh.Host,
			options: def.Ssh.Options,
			executable: struct {
				Bin  string
				Args []string
			}{Bin: def.Ssh.Bin, Args: def.Ssh.Options},
		},
		dir:    def.Dir,
		env:    append(os.Environ(), util.ConvertEnv(def.Env)...),
		def:    &def,
		up:     util.ReadStringsSlice(def.Up),
		down:   util.ReadStringsSlice(def.Down),
		before: util.ReadStringsSlice(def.Before),
		after:  util.ReadStringsSlice(def.After),
	}

	switch c.ctxType {
	case config.CONTEXT_TYPE_CONTAINER:
		switch c.container.provider {
		case config.CONTEXT_CONTAINER_PROVIDER_DOCKER:
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
		case config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
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
		case config.CONTEXT_CONTAINER_PROVIDER_KUBECTL:
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
	case config.CONTEXT_TYPE_REMOTE:
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
			log.Fatal(err)
		}
	}

	if c.executable.Bin == "" {
		setDefaultShell(&c.executable)
	}

	return c, nil
}

func (c *Context) buildLocalCommand(command string) *exec.Cmd {
	cmd := exec.Command(c.executable.Bin, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)
	cmd.Env = c.env
	cmd.Dir = c.dir

	return cmd
}

func (c *Context) buildDockerCommand(command string) *exec.Cmd {
	cmd := exec.Command(c.container.executable.Bin, c.container.executable.Args...)
	cmd.Env = c.env

	switch c.container.provider {
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER:
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
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
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

func (c *Context) buildKubectlCommand(command string) *exec.Cmd {
	cmd := exec.Command(c.container.executable.Bin, c.container.executable.Args...)
	cmd.Env = append(c.env, c.container.env...)

	cmd.Args = append(cmd.Args, "exec", c.container.name)
	cmd.Args = append(cmd.Args, c.container.options...)
	cmd.Args = append(cmd.Args, "--")
	cmd.Args = append(cmd.Args, c.executable.Bin)
	cmd.Args = append(cmd.Args, c.executable.Args...)
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s %s", strings.Join(c.container.env, " "), command))

	return cmd
}

func (c *Context) buildRemoteCommand(command string) *exec.Cmd {
	cmd := exec.Command(c.ssh.executable.Bin, c.ssh.executable.Args...)
	cmd.Args = append(cmd.Args, c.executable.Bin)
	cmd.Args = append(cmd.Args, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)
	cmd.Env = c.env

	return cmd
}

func setDefaultShell(e *util.Executable) {
	e.Bin = "/bin/sh"
	e.Args = []string{"-c"}
}

func (c *Context) Bin() string {
	return c.executable.Bin
}

func (c *Context) Args() []string {
	return c.executable.Args
}

func (c *Context) Env() []string {
	return c.env
}

func (c *Context) WithEnvs(env []string) (*Context, error) {
	def := *c.def
	for _, v := range env {
		kv := strings.Split(v, "=")
		if len(def.Env) == 0 {
			def.Env = make(map[string]string)
		}
		def.Env[kv[0]] = kv[1]
	}

	return BuildContext(def, &config.Get().WilsonConfig)
}

func (c *Context) Up() {
	c.onceUp.Do(func() {
		for _, command := range c.up {
			err := c.runServiceCommand(command)
			if err != nil {
				log.Fatal(err)
			}
		}
	})
}

func (c *Context) Down() {
	c.onceDown.Do(func() {
		for _, command := range c.down {
			err := c.runServiceCommand(command)
			if err != nil {
				log.Error(err)
			}
		}
	})
}

func (c *Context) Before() error {
	for _, command := range c.before {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Context) After() error {
	for _, command := range c.after {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Context) runServiceCommand(command string) error {
	log.Debugf("running service context service command: %s", command)
	ca := strings.Split(command, " ")
	cmd := exec.Command(ca[0], ca[1:]...)
	cmd.Env = c.Env()
	cmd.Dir = util.Getcwd()

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (c *Context) createCommand(command string) *exec.Cmd {
	switch c.ctxType {
	case config.CONTEXT_TYPE_LOCAL:
		return c.buildLocalCommand(command)
	case config.CONTEXT_TYPE_CONTAINER:
		switch c.container.provider {
		case config.CONTEXT_CONTAINER_PROVIDER_DOCKER, config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
			return c.buildDockerCommand(command)
		case config.CONTEXT_CONTAINER_PROVIDER_KUBECTL:
			return c.buildKubectlCommand(command)
		}
	case config.CONTEXT_TYPE_REMOTE:
		return c.buildRemoteCommand(command)
	default:
		log.Fatal("unknown context type")
	}

	return nil
}

func (c *Context) ScheduleForCleanup() {
	c.mu.Lock()
	c.scheduledForCleanup = true
	c.mu.Unlock()

}
