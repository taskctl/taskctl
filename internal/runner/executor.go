package runner

import (
	"bytes"
	"io"
	"os/exec"
)

type CommandExecutor interface {
	Execute(cmd *exec.Cmd) ([]byte, error)
}

type DefaultCommandExecutor struct {
	executed int
}

func NewDefaultCommandExecutor() *DefaultCommandExecutor {
	return &DefaultCommandExecutor{}
}

func (d *DefaultCommandExecutor) Execute(cmd *exec.Cmd) ([]byte, error) {
	defer d.inc()

	var buf bytes.Buffer

	cmd.Stdout = io.MultiWriter(&buf, cmd.Stdout)
	cmd.Stderr = io.MultiWriter(&buf, cmd.Stderr)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	return buf.Bytes(), err
}

func (d *DefaultCommandExecutor) inc() {
	d.executed += 1
}
