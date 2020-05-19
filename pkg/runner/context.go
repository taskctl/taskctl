package runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/taskctl/taskctl/pkg/executor"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/utils"

	"github.com/sirupsen/logrus"
)

type ExecutionContext struct {
	Executable *utils.Binary
	Dir        string
	Env        variables.Container
	Variables  variables.Container

	up     []string
	down   []string
	before []string
	after  []string

	startupError error

	onceUp   sync.Once
	onceDown sync.Once
	mu       sync.Mutex
}

func NewExecutionContext(executable *utils.Binary, dir string, env variables.Container, up, down, before, after []string) *ExecutionContext {
	c := &ExecutionContext{
		Executable: executable,
		Env:        env,
		Dir:        dir,
		up:         up,
		down:       down,
		before:     before,
		after:      after,
		Variables:  variables.NewVariables(),
	}

	return c
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
	ex, err := executor.NewDefaultExecutor()
	if err != nil {
		return err
	}

	out, err := ex.Execute(context.Background(), &executor.Job{
		Command: command,
		Dir:     c.Dir,
		Env:     c.Env,
		Vars:    c.Variables,
	})
	if err != nil {
		if _, ok := executor.IsExitStatus(err); ok {
			return fmt.Errorf("%v\n%s\n%s\n", err, out, err.Error())
		}
		return err
	}

	return nil
}

func DefaultContext() *ExecutionContext {
	return &ExecutionContext{
		Env: variables.NewVariables(),
	}
}
