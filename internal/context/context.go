package context

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/taskctl/taskctl/internal/task"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/internal/util"
)

type ExecutionContext struct {
	ScheduledForCleanup bool

	executable util.Executable
	env        []string
	dir        string
	up         []string
	down       []string
	before     []string
	after      []string

	startupError error

	onceUp   sync.Once
	onceDown sync.Once
	mu       sync.Mutex
}

func NewExecutionContext(executable util.Executable, dir string, env, up, down, before, after []string) *ExecutionContext {
	c := &ExecutionContext{
		executable: executable,
		env:        env,
		dir:        dir,
		up:         up,
		down:       down,
		before:     before,
		after:      after,
	}

	return c
}

func (c *ExecutionContext) Bin() string {
	return c.executable.Bin
}

func (c *ExecutionContext) Args() []string {
	return c.executable.Args
}

func (c *ExecutionContext) Env() []string {
	return c.env
}

func (c *ExecutionContext) Up() error {
	c.onceUp.Do(func() {
		for _, command := range c.up {
			err := c.runServiceCommand(command)
			if err != nil {
				c.mu.Lock()
				c.startupError = err
				c.mu.Unlock()
				logrus.Errorf("context startup error: %s", err)
			}
		}
	})

	return c.startupError
}

func (c *ExecutionContext) Down() {
	c.onceDown.Do(func() {
		for _, command := range c.down {
			err := c.runServiceCommand(command)
			if err != nil {
				logrus.Errorf("context cleanup error: %s", err)
			}
		}
	})
}

func (c *ExecutionContext) Before() error {
	for _, command := range c.before {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) After() error {
	for _, command := range c.after {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) runServiceCommand(command string) (err error) {
	logrus.Debugf("running context service command: %s", command)
	ca := strings.Split(command, " ")
	cmd := exec.Command(ca[0], ca[1:]...)
	cmd.Env = c.Env()
	cmd.Dir, err = os.Getwd()
	if err != nil {
		return err
	}

	out, err := cmd.Output()
	if err != nil {
		if exerr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("%v\n%s\n%s\n", err, out, exerr.Stderr)
		} else {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) BuildCommand(ctx context.Context, command string, t *task.Task) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, c.executable.Bin, c.executable.Args...)
	cmd.Args = append(cmd.Args, command)
	cmd.Env = c.env
	cmd.Dir = c.dir

	if cmd == nil {
		return nil, errors.New("failed to build command")
	}

	if t != nil && t.Dir != "" {
		cmd.Dir = t.Dir
	}

	return cmd, nil
}

func (c *ExecutionContext) ScheduleForCleanup() {
	c.mu.Lock()
	c.ScheduledForCleanup = true
	c.mu.Unlock()
}
