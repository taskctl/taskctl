package runner

import (
	"errors"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/util"
	"os"
	"strings"
)

type Context struct {
	Type       string
	executable util.Executable
	env        []string
	def        *config.ContextConfig
}

type contextBuilder struct {
	def *config.ContextConfig
	w   *config.WilsonConfig
}

func BuildContext(def *config.ContextConfig) (*Context, error) {
	contextBuilder := &contextBuilder{def: def, w: &config.Get().WilsonConfig}

	return contextBuilder.build()
}

func (cb *contextBuilder) build() (*Context, error) {
	c := &Context{
		Type: cb.def.Type,
		executable: util.Executable{
			Bin:  cb.def.Executable.Bin,
			Args: make([]string, 0),
		},
		env: append(os.Environ(), util.ConvertEnv(cb.def.Env)...),
		def: cb.def,
	}

	switch cb.def.Type {
	case config.CONTEXT_TYPE_LOCAL:
		if c.executable.Bin != "" {
			break
		}

		if cb.w.Shell.Bin != "" {
			c.executable.Bin = cb.w.Shell.Bin
			c.executable.Args = cb.w.Shell.Args
		} else {
			setDefaultShell(&c.executable)
		}

	case config.CONTEXT_TYPE_CONTAINER:
		switch cb.def.Container.Provider {
		case config.CONTEXT_CONTAINER_PROVIDER_DOCKER, config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
			err := cb.buildDockerContext(c)
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

func (cb *contextBuilder) buildDockerContext(c *Context) error {
	bin := cb.def.Executable.Bin
	if bin == "" {
		setDefaultShell(&cb.def.Executable)
		bin = cb.def.Executable.Bin
	}
	args := cb.def.Executable.Args

	switch cb.def.Container.Provider {
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER:
		if cb.w.Docker.Bin != "" {
			c.executable.Bin = cb.w.Docker.Bin
		} else {
			c.executable.Bin = "docker"
		}

		c.executable.Args = cb.w.Docker.Args
		c.executable.Args = append(c.executable.Args, cb.def.Container.Options...)

		if cb.def.Container.Exec {
			c.executable.Args = append(c.executable.Args, "exec")
			c.executable.Args = append(c.executable.Args, cb.def.Container.Name)
		} else {
			c.executable.Args = append(c.executable.Args, "run", "--rm")
			c.executable.Args = append(c.executable.Args, cb.def.Container.Image)
		}

		for _, v := range util.ConvertEnv(cb.def.Container.Env) {
			c.executable.Args = append(c.executable.Args, "-e", v)
		}
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
		if cb.w.DockerCompose.Bin != "" {
			c.executable.Bin = cb.w.DockerCompose.Bin
		} else {
			c.executable.Bin = "docker-compose"
		}

		c.executable.Args = cb.w.DockerCompose.Args
		c.executable.Args = append(c.executable.Args, cb.def.Container.Options...)

		if cb.def.Container.Exec {
			c.executable.Args = append(c.executable.Args, "exec", "-T")
		} else {
			c.executable.Args = append(c.executable.Args, "run", "--rm")
		}

		for _, v := range util.ConvertEnv(cb.def.Container.Env) {
			c.executable.Args = append(c.executable.Args, "-e", v)
		}

		c.executable.Args = append(c.executable.Args, cb.def.Container.Name)
	}

	c.executable.Args = append(c.executable.Args, bin)
	c.executable.Args = append(c.executable.Args, args...)

	return nil
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

	return BuildContext(&def)
}
