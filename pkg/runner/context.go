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

	"github.com/Ensono/taskctl/pkg/executor"
	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/sirupsen/logrus"
)

var (
	// define a list of environment variable names that are not permitted
	invalidEnvVarKeys = []string{
		"",                              //skip any empty key names
		`!::`, `=::`, `::=::`, `::=::\`, // this is found in a cygwin environment
	}
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
func (c *ExecutionContext) GenerateEnvfile() error {
	// return an error if the include and exclude have both been specified
	if len(c.Envfile.Exclude) > 0 && len(c.Envfile.Include) > 0 {
		return fmt.Errorf("include and exclude lists are mutually exclusive, %w", ErrMutuallyExclusiveVarSet)
	}

	// create a string builder object to hold all of the lines that need to be written out to
	// the resultant file
	builder := []string{}

	// iterate around all of the environment variables and add the selected ones to the builder
	for varName, varValue := range c.Env.Map() {
		// check to see if the env matches an invalid variable, if it does
		// move onto the next item in the  loop
		if slices.Contains(invalidEnvVarKeys, varName) {
			logrus.Warnf("Skipping invalid env var: %s=%v\n'%s' is not a valid key", varName, varValue, varName)
			continue
		}

		// iterate around the modify options to see if the name needs to be
		// modified at all
		for _, modify := range c.Envfile.Modify {

			// use the pattern to determine if the string has been identified
			// this assumes 1 capture group so this will be used as the name to transform
			re := regexp.MustCompile(modify.Pattern)
			match := re.FindStringSubmatch(varName)
			if len(match) > 0 {

				keyword := match[re.SubexpIndex("keyword")]
				varname := match[re.SubexpIndex("varname")]

				// perform the operation on the varname
				switch modify.Operation {
				case "lower":
					varname = strings.ToLower(varname)
				case "upper":
					varname = strings.ToUpper(varname)
				}

				// Build up the name
				varName = fmt.Sprintf("%s%s", keyword, varname)

				break
			}
		}

		// determine if the variable should be included or excluded
		// ShouldExclude will be true if any varName
		shouldExclude := slices.ContainsFunc(c.Envfile.Exclude, func(v string) bool {
			return strings.HasPrefix(varName, v)
		})

		shouldInclude := true
		if len(c.Envfile.Include) > 0 {
			shouldInclude = slices.ContainsFunc(c.Envfile.Include, func(v string) bool {
				return strings.HasPrefix(varName, v)
			})
		}

		// if the variable should excluded or not explicitly included then move onto the next variable
		if shouldExclude || !shouldInclude {
			continue
		}

		// sanitize variable values from newline and space characters
		// replace any newline characters with a space, this is to prevent multiline variables being passed in
		// quote the value if it has spaces in it
		// TODO: this should be discussed? why?
		// supplied values should be left in-tact?
		//
		// value := strings.NewReplacer("\n", c.Envfile.ReplaceChar, `\s`, "").Replace(varValue)

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
		c.Quote = quote
	}
}
