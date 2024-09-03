package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ensono/taskctl/pkg/executor"

	"github.com/ensono/taskctl/pkg/variables"

	"github.com/ensono/taskctl/pkg/utils"

	"github.com/sirupsen/logrus"
)

// ExecutionContext allow you to set up execution environment, variables, binary which will run your task, up/down commands etc.
type ExecutionContext struct {
	Executable *utils.Binary
	Dir        string
	Env        variables.Container
	Envfile    *utils.Envfile
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
func NewExecutionContext(executable *utils.Binary, dir string, env variables.Container, envfile *utils.Envfile, up, down, before, after []string, options ...ExecutionContextOption) *ExecutionContext {
	c := &ExecutionContext{
		Executable: executable,
		Env:        env,
		Envfile:    envfile,
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

func (c *ExecutionContext) GenerateEnvfile() error {

	// only generate the file if it has been explicitly asked for
	if !c.Envfile.Generate {
		return nil
	}

	// set default values
	if c.Envfile.Path == "" {
		c.Envfile.Path = "envfile"
	}

	if c.Envfile.ReplaceChar == "" {
		c.Envfile.ReplaceChar = " "
	}

	// return an error if the include and exclude have both been specified
	if len(c.Envfile.Exclude) > 0 && len(c.Envfile.Include) > 0 {
		err := errors.New("include and exclude lists are mutually exclusive")
		return err
	}

	// determine the path to the envfile
	// if it is not absolute then prepare the current dir to it
	isAbsolute := filepath.IsAbs(c.Envfile.Path)
	if !isAbsolute {

		// get the current working directory
		cwd, err := os.Getwd()

		if err != nil {
			return err
		}

		c.Envfile.Path = filepath.Join(cwd, c.Envfile.Path)
	}

	// create a string builder object to hold all of the lines that need to be written out to
	// the resultant file
	builder := strings.Builder{}
	spacePattern := regexp.MustCompile(`\s`)

	// iterate around all of the environment variables and add the selected ones to the builder
	for _, env := range os.Environ() {

		// split the environment variable using = as the delimiter
		// this is so that newlines can be surpressed
		parts := strings.SplitN(env, "=", 2)

		// Get the name of the variable
		name := parts[0]

		// determine if the variable should be included or excluded
		shouldExclude := utils.SliceContains(c.Envfile.Exclude, name)

		shouldInclude := true
		if len(c.Envfile.Include) > 0 {
			shouldInclude = utils.SliceContains(c.Envfile.Include, name)
		}

		// if the variable should excluded or not explicitly included then move onto the next variable
		if shouldExclude || !shouldInclude {
			continue
		}

		// replace any newline characters with a space, this is to prevent multiline variables being passed in
		value := strings.Replace(parts[1], "\n", c.Envfile.ReplaceChar, -1)

		// quote the value if it has spaces in it
		if spacePattern.MatchString(value) && c.Envfile.Quote {
			value = fmt.Sprintf("\"%s\"", value)
		}

		// Add the name and the value to the string builder
		builder.WriteString(fmt.Sprintf("%s=%s\n", name, value))
	}

	// get the full output from the string builder
	output := builder.String()

	// write the output to the file
	if err := os.WriteFile(c.Envfile.Path, []byte(output), 0666); err != nil {
		logrus.Fatalf("Error writing out file: %s\n", err.Error())
	}

	logrus.Debug(output)

	// delay the ongoing execution of taskctl if a value has been set
	if c.Envfile.Delay > 0 {
		time.Sleep(time.Duration(c.Envfile.Delay) * time.Millisecond)
	}

	return nil
}

func (c *ExecutionContext) runServiceCommand(command string) (err error) {
	logrus.Debugf("running context service command: %s", command)
	ex, err := executor.NewDefaultExecutor(nil, nil, nil)
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
		Envfile:   &utils.Envfile{},
		Variables: variables.NewVariables(),
	}
}

// WithQuote is functional option to set Quote for ExecutionContext
func WithQuote(quote string) ExecutionContextOption {
	return func(c *ExecutionContext) {
		c.Quote = quote
	}
}
