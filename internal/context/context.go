package context

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/taskctl/taskctl/internal/utils"

	"github.com/sirupsen/logrus"
)

type ExecutionContext struct {
	Executable utils.Executable
	Env        []string
	Dir        string

	up     []string
	down   []string
	before []string
	after  []string

	startupError error

	onceUp   sync.Once
	onceDown sync.Once
	mu       sync.Mutex
}

func NewExecutionContext(executable utils.Executable, dir string, env, up, down, before, after []string) *ExecutionContext {
	c := &ExecutionContext{
		Executable: executable,
		Env:        env,
		Dir:        dir,
		up:         up,
		down:       down,
		before:     before,
		after:      after,
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
	ca := strings.Split(command, " ")
	cmd := exec.Command(ca[0], ca[1:]...)
	cmd.Env = c.Env
	if c.Dir != "" {
		cmd.Dir = c.Dir
	} else {
		cmd.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
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
