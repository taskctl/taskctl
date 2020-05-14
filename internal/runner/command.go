package runner

import (
	"context"

	"github.com/taskctl/taskctl/internal/variables"

	"os"
	"os/exec"

	taskctx "github.com/taskctl/taskctl/internal/context"
	"github.com/taskctl/taskctl/internal/utils"
)

func createCommand(ctx context.Context, executionCtx *taskctx.ExecutionContext, command string, envs ...variables.Container) (*exec.Cmd, error) {
	env := executionCtx.Env
	for _, v := range envs {
		env = env.Merge(v)
	}

	var cmd *exec.Cmd
	if executionCtx.Executable != nil {
		cmd = exec.CommandContext(ctx, executionCtx.Executable.Bin, executionCtx.Executable.Args...)
		cmd.Args = append(cmd.Args, command)
	} else {
		cmd = exec.CommandContext(ctx, command)
	}

	cmd.Env = utils.ConvertEnv(env.Map())
	cmd.Dir = executionCtx.Dir

	if cmd.Dir == "" {
		var err error
		cmd.Dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}
