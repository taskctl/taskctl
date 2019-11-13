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
	case config.CONTEXT_TYPE_DOCKER_COMPOSE:
		bin := def.Executable.Bin
		c.Executable.Bin = "docker-compose" // todo: move to config
		if def.ComposeService.File != "" {
			c.Executable.Args = append(c.Executable.Args, "--file", def.ComposeService.File)
		}

		if def.ComposeService.Transient {
			c.Executable.Args = append(c.Executable.Args, "run", "--rm")
		} else {
			// todo: ensure context running
			c.Executable.Args = append(c.Executable.Args, "exec")
		}

		c.Executable.Args = append(c.Executable.Args, def.ComposeService.Name)
		c.Executable.Args = append(c.Executable.Args, bin)
		c.Executable.Args = append(c.Executable.Args, def.Executable.Args...)
	default:
		return nil, errors.New("context type not implemented")
	}

	if c.Executable.Bin == "" {
		return nil, errors.New("invalid context executable")
	}

	if def.ComposeService.StartupCommand != "" {
		// todo: start context
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

func (c *Context) Cleanup() {
	// todo: cleanup
}
