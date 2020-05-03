package context

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/taskctl/taskctl/pkg/config"
	"github.com/taskctl/taskctl/pkg/util"
)

func buildContainerContext(def *config.ContextDefinition, cfg *config.Config, c *ExecutionContext) {
	c.container = container{
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
	}

	switch c.container.provider {
	case config.ContextContainerProviderDocker:
		if c.container.executable.Bin == "" {
			if cfg.Docker.Bin != "" {
				c.container.executable.Bin = cfg.Docker.Bin
			} else {
				c.container.executable.Bin = "docker"
			}
		}
		if len(c.container.executable.Args) == 0 {
			c.container.executable.Args = cfg.Docker.Args
		}
	case config.ContextContainerProviderDockerCompose:
		if c.container.executable.Bin == "" {
			if cfg.DockerCompose.Bin != "" {
				c.container.executable.Bin = cfg.DockerCompose.Bin
			} else {
				c.container.executable.Bin = "docker-compose"
			}
		}
		if len(c.container.executable.Args) == 0 {
			c.container.executable.Args = cfg.DockerCompose.Args
		}
	case config.ContextContainerProviderKubectl:
		if c.container.executable.Bin == "" {
			if cfg.Kubectl.Bin != "" {
				c.container.executable.Bin = cfg.Kubectl.Bin
			} else {
				c.container.executable.Bin = "kubectl"
			}
		}

		if len(c.container.executable.Args) == 0 {
			c.container.executable.Args = cfg.Kubectl.Args
		}
	}
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
