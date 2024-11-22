package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/executor"
	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/sirupsen/logrus"
)

var (
	// define a list of environment variable names that are not permitted
	invalidEnvVarKeys = []string{
		"",                              // skip any empty key names
		`!::`, `=::`, `::=::`, `::=::\`, // this is found in a cygwin environment
	}
)

// ExecutionContext allow you to set up execution environment, variables, binary which will run your task, up/down commands etc.
type ExecutionContext struct {
	Executable *utils.Binary
	container  *utils.Container
	Dir        string
	Env        *variables.Variables
	Envfile    *utils.Envfile
	Variables  *variables.Variables
	// Quote character to use around a command
	// when passed to another executable, e.g. docker
	Quote string

	up     []string
	down   []string
	before []string
	after  []string

	startupError error

	onceUp   sync.Once
	onceDown sync.Once
	mu       *sync.Mutex
}

// ExecutionContextOption is a functional option to configure ExecutionContext
type ExecutionContextOption func(c *ExecutionContext)

// NewExecutionContext creates new ExecutionContext instance
func NewExecutionContext(executable *utils.Binary, dir string,
	env *variables.Variables, envfile *utils.Envfile, up, down, before, after []string,
	options ...ExecutionContextOption) *ExecutionContext {
	c := &ExecutionContext{
		// mu is a pointer to a mutex
		// so that it's shared across all
		// the instances that are using the given ExecutionContext
		mu:        &sync.Mutex{},
		Variables: variables.NewVariables(),
	}

	for _, o := range options {
		o(c)
	}

	c.Executable = executable
	c.Env = env
	c.Envfile = envfile
	c.Dir = dir
	c.up = up
	c.down = down
	c.before = before
	c.after = after

	return c
}

func WithContainerOpts(containerOpts *utils.Container) ExecutionContextOption {
	return func(c *ExecutionContext) {
		c.container = containerOpts
		// add additional closed properties
	}
}

func (c *ExecutionContext) Container() *utils.Container {
	return c.container
}

// StartUpError reports whether an error exists on startUp
func (c *ExecutionContext) StartupError() error {
	return c.startupError
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

var ErrMutuallyExclusiveVarSet = errors.New("mutually exclusive vars have been set")

// GenerateEnvfile processes env and other supplied variables
// writes them to a `.taskctl` folder in a current directory
// the file names are generated using the `generated_{Task_Name}_{UNIX_timestamp}.env`.
//
// Note: it will create the directory
func (c *ExecutionContext) GenerateEnvfile(env *variables.Variables) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// return an error if the include and exclude have both been specified
	if len(c.Envfile.Exclude) > 0 && len(c.Envfile.Include) > 0 {
		return fmt.Errorf("include and exclude lists are mutually exclusive, %w", ErrMutuallyExclusiveVarSet)
	}

	// create a string builder object to hold all of the lines that need to be written out to
	// the resultant file
	builder := []string{}
	// iterate through all of the environment variables and add the selected ones to the builder
	// env container at this point should already include all the merged variables by precedence
	// TODO: if envfile path was provided we should merge it in with Env and inject as a whole into the container
	for varName, varValue := range env.Map() {
		// check to see if the env matches an invalid variable, if it does
		// move onto the next item in the  loop
		if slices.Contains(invalidEnvVarKeys, varName) {
			logrus.Warnf("Skipping invalid env var: %s=%v\n'%s' is not a valid key", varName, varValue, varName)
			continue
		}

		varName = c.modifyName(varName)
		// determine if the variable should be included or excluded
		if c.includeExcludeSkip(varName) {
			continue
		}

		// sanitize variable values from newline and space characters
		// replace any newline characters with a space, this is to prevent multiline variables being passed in
		// quote the value if it has spaces in it
		// Add the name and the value to the string builder
		envstr := fmt.Sprintf("%s=%s", varName, varValue)
		builder = append(builder, envstr)
		logrus.Debug(envstr)
	}

	// get the full output from the string builder
	// write the output to the file
	if err := os.MkdirAll(filepath.Dir(c.Envfile.Path), 0700); err != nil {
		logrus.Fatalf("Error creating parent directory for artifacts: %s\n", err.Error())
	}

	return os.WriteFile(c.Envfile.Path, []byte(strings.Join(builder, "\n")), 0700)
}

func (c *ExecutionContext) includeExcludeSkip(varName string) bool {
	// set var name to lower to ensure case-insensitive comparison
	varName = strings.ToLower(varName)
	// ShouldExclude will be true if any varName
	shouldExclude := slices.ContainsFunc(c.Envfile.Exclude, func(v string) bool {
		return strings.HasPrefix(varName, strings.ToLower(v))
	})

	shouldInclude := true
	if len(c.Envfile.Include) > 0 {
		shouldInclude = slices.ContainsFunc(c.Envfile.Include, func(v string) bool {
			return strings.HasPrefix(varName, strings.ToLower(v))
		})
	}

	// if the variable should excluded or not explicitly included then move onto the next variable
	return shouldExclude || !shouldInclude
}

func (c *ExecutionContext) modifyName(varName string) string {
	// iterate around the modify options to see if the name needs to be
	// modified at all
	for _, modify := range c.Envfile.Modify {

		// use the pattern to determine if the string has been identified
		// this assumes 1 capture group so this will be used as the name to transform
		re := regexp.MustCompile(modify.Pattern)
		match := re.FindStringSubmatch(varName)
		if len(match) > 0 {

			keyword := match[re.SubexpIndex("keyword")]
			matchedVarName := match[re.SubexpIndex("varname")]

			// perform the operation on the varname
			switch modify.Operation {
			case "lower":
				matchedVarName = strings.ToLower(matchedVarName)
			case "upper":
				matchedVarName = strings.ToUpper(matchedVarName)
			}
			// Build up the name
			return fmt.Sprintf("%s%s", keyword, matchedVarName)
		}
	}
	return varName
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
	// the default context still needs access to global env variables
	return NewExecutionContext(nil, "",
		variables.FromMap(utils.ConvertFromEnv(os.Environ())),
		&utils.Envfile{},
		[]string{},
		[]string{},
		[]string{},
		[]string{},
	)
}

// WithQuote is functional option to set Quote for ExecutionContext
func WithQuote(quote string) ExecutionContextOption {
	return func(c *ExecutionContext) {
		c.Quote = "'"
		if quote != "" {
			c.Quote = quote
		}
	}
}
