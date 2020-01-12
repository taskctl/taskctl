package context

import (
	"context"
	"fmt"
	"github.com/trntv/wilson/pkg/builder"
	"github.com/trntv/wilson/pkg/util"
	"os/exec"
)

func buildRemoteContext(def *builder.ContextDefinition, wcfg *builder.WilsonConfigDefinition, c *ExecutionContext) {
	c.ssh = ssh{
		user:    def.SSH.User,
		host:    def.SSH.Host,
		options: def.SSH.Options,
		executable: util.Executable{
			Bin:  def.SSH.Bin,
			Args: def.SSH.Options,
		},
	}
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

func (c *ExecutionContext) buildRemoteCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.ssh.executable.Bin, c.ssh.executable.Args...)
	cmd.Env = c.env
	cmd.Dir = c.dir

	cmd.Args = append(cmd.Args, c.executable.Bin)
	cmd.Args = append(cmd.Args, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)

	return cmd
}
