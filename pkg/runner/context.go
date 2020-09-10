package runner

import (
	"context"
	"sync"

	"github.com/taskctl/taskctl/pkg/executor"

	"github.com/taskctl/taskctl/pkg/variables"

	"github.com/taskctl/taskctl/pkg/utils"

	"github.com/sirupsen/logrus"
)

// ExecutionContext allow you to set up execution environment, variables, binary which will run your task, up/down commands etc.
type ExecutionContext struct {
	Executable *utils.Binary
	Dir        string
	Env        variables.Container
	Variables  variables.Container
	Quote      string

	up     []string
	down   []string
	before []string
	after  []string

	startupError error

	onceUp   sync.Once
	onceDown sync.Once
	mu       sync.Mutex
}

// ExecutionContextOption is a functional option to configure ExecutionContext
type ExecutionContextOption func(c *ExecutionContext)

// NewExecutionContext creates new ExecutionContext instance
func NewExecutionContext(executable *utils.Binary, dir string, env variables.Container, up, down, before, after []string, options ...ExecutionContextOption) *ExecutionContext {
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

	for _, o := range options {
		o(c)
	}

	return c
}

// Up executes tasks defined to run once before first usage of the context
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

// Down executes tasks defined to run once after last usage of the context
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

// Before executes tasks defined to run before every usage of the context
func (c *ExecutionContext) Before() error {
	for _, command := range c.before {
		err := c.runServiceCommand(command)
		if err != nil {
			return err
		}
	}

	return nil
}

// After executes tasks defined to run after every usage of the context
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
		if out != nil {
			logrus.Warning(string(out))
		}

		return err
	}

	return nil
}

// DefaultContext creates default ExecutionContext instance
func DefaultContext() *ExecutionContext {
	return &ExecutionContext{
		Env:       variables.NewVariables(),
		Variables: variables.NewVariables(),
	}
}

// WithQuote is functional option to set Quote for ExecutionContext
func WithQuote(quote string) ExecutionContextOption {
	return func(c *ExecutionContext) {
		c.Quote = quote
	}
}
