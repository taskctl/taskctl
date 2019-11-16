package runner

import (
	"errors"
	"github.com/trntv/wilson/pkg/config"
	"os"
	"strings"
)

type Context struct {
	Type       string
	container  Container
	executable Executable
	env        []string
	def        *config.ContextConfig
}

type Executable struct {
	Bin  string
	Args []string
}

type Container struct {
	Provider string
	Name     string
	Run      bool
}

func BuildContext(def *config.ContextConfig) (*Context, error) {
	c := &Context{
		Type:      def.Type,
		container: Container{},
		executable: Executable{
			Bin:  def.Executable.Bin,
			Args: make([]string, 0),
		},
		env: append(os.Environ(), config.ConvertEnv(def.Env)...),
		def: def,
	}

	switch def.Type {
	case config.CONTEXT_TYPE_LOCAL:
		if c.executable.Bin == "" {
			c.executable.Bin = "/bin/sh" // todo: move to config
			c.executable.Args = []string{"-c"}
		}
	case config.CONTEXT_TYPE_CONTAINER:
		switch def.Container.Provider {
		case config.CONTEXT_CONTAINER_PROVIDER_DOCKER, config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
			err := buildDockerContext(def, c)
			if err != nil {
				return nil, err
			}
		case config.CONTEXT_CONTAINER_PROVIDER_KUBECTL:
			return nil, errors.New("kubectl provider not implemented")
		}

	default:
		return nil, errors.New("context type not implemented")
	}

	if c.executable.Bin == "" {
		return nil, errors.New("invalid context config")
	}

	return c, nil
}

func buildDockerContext(def *config.ContextConfig, c *Context) error {
	bin := def.Executable.Bin
	if bin == "" {
		defaultShell(&def.Executable)
		bin = def.Executable.Bin
	}
	args := def.Executable.Args

	switch def.Container.Provider {
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER:
		c.executable.Bin = "docker" // todo: move to config
		c.executable.Args = def.Container.Options

		if def.Container.Exec {
			c.executable.Args = append(c.executable.Args, "exec")
			c.executable.Args = append(c.executable.Args, def.Container.Name)
		} else {
			c.executable.Args = append(c.executable.Args, "run", "--rm")
			c.executable.Args = append(c.executable.Args, def.Container.Image)
		}

		for _, v := range config.ConvertEnv(def.Env) {
			c.executable.Args = append(c.executable.Args, "-e", v)
		}
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
		c.executable.Bin = "docker-compose" // todo: move to config
		c.executable.Args = def.Container.Options

		if def.Container.Exec {
			c.executable.Args = append(c.executable.Args, "exec", "-T")
		} else {
			c.executable.Args = append(c.executable.Args, "run", "--rm")
		}

		for _, v := range config.ConvertEnv(def.Env) {
			c.executable.Args = append(c.executable.Args, "-e", v)
		}

		c.executable.Args = append(c.executable.Args, def.Container.Name)
	}

	c.executable.Args = append(c.executable.Args, bin)
	c.executable.Args = append(c.executable.Args, args...)

	return nil
}

func defaultShell(e *config.Executable) {
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
	return BuildContext(&def)
}
