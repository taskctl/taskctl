package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/taskctl/taskctl/executor"

	"github.com/taskctl/taskctl/variables"
)

// Binary is a structure for storing binary file path and arguments that should be passed on binary's invocation
type Binary struct {
	Bin  string
	Args []string
}

// ExecutionContext allow you to set up execution environment, variables, binary which will run your task, up/down commands etc.
type ExecutionContext struct {
	Executable *Binary
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
func NewExecutionContext(executable *Binary, dir string, env variables.Container, up, down, before, after []string, options ...ExecutionContextOption) *ExecutionContext {
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
func (c *ExecutionContext) Up(ctx context.Context) error {
	c.onceUp.Do(func() {
		for _, command := range c.up {
			err := c.runServiceCommand(ctx, command)
			if err != nil {
				c.mu.Lock()
				c.startupError = err
				c.mu.Unlock()
				slog.Error(fmt.Sprintf("context startup error: %s", err.Error()))
			}
		}
	})

	return c.startupError
}

// Down executes tasks defined to run once after last usage of the context
func (c *ExecutionContext) Down() {
	c.onceDown.Do(func() {
		for _, command := range c.down {
			// Cleanup must run even when the run was cancelled, so it uses a
			// fresh background context rather than the (possibly cancelled) one.
			err := c.runServiceCommand(context.Background(), command)
			if err != nil {
				slog.Error(fmt.Sprintf("context cleanup error: %s", err.Error()))
			}
		}
	})
}

// Before executes tasks defined to run before every usage of the context
func (c *ExecutionContext) Before(ctx context.Context) error {
	for _, command := range c.before {
		err := c.runServiceCommand(ctx, command)
		if err != nil {
			return err
		}
	}

	return nil
}

// After executes tasks defined to run after every usage of the context
func (c *ExecutionContext) After() error {
	for _, command := range c.after {
		// After runs during task teardown, so it uses a background context to
		// stay independent of the run's cancellation.
		err := c.runServiceCommand(context.Background(), command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ExecutionContext) runServiceCommand(ctx context.Context, command string) (err error) {
	slog.Debug(fmt.Sprintf("running context service command: %s", command))
	ex, err := executor.NewDefaultExecutor(nil, nil, nil)
	if err != nil {
		return err
	}

	out, err := ex.Execute(ctx, &executor.Job{
		Command: command,
		Dir:     c.Dir,
		Env:     c.Env,
		Vars:    c.Variables,
	})
	if err != nil {
		if out != nil {
			slog.Warn(string(out))
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
