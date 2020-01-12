package context

import (
	"context"
	"os/exec"
)

func (c *ExecutionContext) buildLocalCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.executable.Bin, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)
	cmd.Env = c.env
	cmd.Dir = c.dir

	return cmd
}
