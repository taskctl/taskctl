package runner

import (
	"errors"
	"github.com/trntv/wilson/pkg/config"
	"os"
)

func BuildContext(def *config.ContextConfig) (*Context, error) {
	c := &Context{
		Type: def.Type,
		Executable: Executable{
			Bin:  def.Executable.Bin,
			Args: make([]string, 0),
		},
		Container: Container{},
		Env:       append(os.Environ(), config.ConvertEnv(def.Env)...),
	}

	switch def.Type {
	case config.CONTEXT_TYPE_LOCAL:
		if c.Executable.Bin == "" {
			c.Executable.Bin = "/bin/sh" // todo: move to config
			c.Executable.Args = []string{"-c"}
		}
	case config.CONTEXT_TYPE_CONTAINER:
		err := buildContainerContext(def, c)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("context type not implemented")
	}

	if c.Executable.Bin == "" {
		return nil, errors.New("invalid context config")
	}

	return c, nil
}

type Context struct {
	Type       string
	Executable Executable
	Container  Container
	Env        []string
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

func buildContainerContext(def *config.ContextConfig, c *Context) error {
	switch def.Container.Provider {
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER:
		bin := def.Executable.Bin
		if bin == "" {
			defaultShell(&def.Executable)
			bin = def.Executable.Bin
		}

		c.Executable.Bin = "docker" // todo: move to config
		c.Executable.Args = append(c.Executable.Args, def.Container.Options...)

		if def.Container.Exec {
			c.Executable.Args = append(c.Executable.Args, "exec")
			c.Executable.Args = append(c.Executable.Args, def.Container.Name)
		} else {
			c.Executable.Args = append(c.Executable.Args, "run", "--rm")
			c.Executable.Args = append(c.Executable.Args, def.Container.Image)
		}

		c.Executable.Args = append(c.Executable.Args, bin)
		c.Executable.Args = append(c.Executable.Args, def.Executable.Args...)
	case config.CONTEXT_CONTAINER_PROVIDER_DOCKER_COMPOSE:
		bin := def.Executable.Bin
		if bin == "" {
			defaultShell(&def.Executable)
			bin = def.Executable.Bin
		}

		c.Executable.Bin = "docker-compose" // todo: move to config
		c.Executable.Args = append(c.Executable.Args, def.Container.Options...)

		if def.Container.Exec {
			c.Executable.Args = append(c.Executable.Args, "exec", "-T")
		} else {
			c.Executable.Args = append(c.Executable.Args, "run", "--rm")
		}

		c.Executable.Args = append(c.Executable.Args, def.Container.Name)
		c.Executable.Args = append(c.Executable.Args, bin)
		c.Executable.Args = append(c.Executable.Args, def.Executable.Args...)
	case config.CONTEXT_CONTAINER_PROVIDER_KUBECTL:
		return errors.New("kubectl provider not implemented")
	}

	return nil
}

func defaultShell(e *config.Executable) {
	e.Bin = "/bin/sh"
	e.Args = []string{"-c"}
}

func (c *Context) Cleanup() {
	// todo: cleanup
}
