package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/spf13/cobra"
)

var cancel = make(chan struct{})

// ensure this initialised from Viper
var conf *config.Config

var taskRunner *runner.TaskRunner

// Arg munging
var (
	taskOrPipelineName string                    = ""
	pipelineName       *scheduler.ExecutionGraph = nil
	taskName           *task.Task                = nil
	argsList           []string                  = nil
)

var (
	runCmd = &cobra.Command{
		Use:     "run",
		Aliases: []string{"r", "fetch", "get"},
		Short:   `runs <pipeline or task>`,
		Long: `taskctl run pipeline1
taskctl run task1`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return buildTaskRunner(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTarget(taskRunner)
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			postRunReset()
			return nil
		},
	}
)

func init() {
	TaskCtlCmd.AddCommand(runCmd)
}

// postRunReset is a test helper function to clear any set values
func postRunReset() {
	// cancel = nil
	conf = nil
	taskRunner = nil
	taskOrPipelineName = ""
	pipelineName = nil
	taskName = nil
	argsList = nil
	debug = false
	cfg = ""
	output = ""
	raw = false
	cockpit = false
	quiet = false
	variableSet = nil
	dryRun = false
	summary = false
}

func runTarget(taskRunner *runner.TaskRunner) (err error) {

	if pipelineName != nil {
		if err := runPipeline(pipelineName, taskRunner, conf.Summary); err != nil {
			return fmt.Errorf("pipeline %s failed: %w", taskOrPipelineName, err)
		}
		return nil
	}

	if taskName != nil {
		if err := runTask(taskName, taskRunner); err != nil {
			return fmt.Errorf("task %s failed: %w", taskOrPipelineName, err)
		}
	}

	return nil
}

func runPipeline(g *scheduler.ExecutionGraph, taskRunner *runner.TaskRunner, summary bool) error {
	sd := scheduler.NewScheduler(taskRunner)
	go func() {
		<-cancel
		sd.Cancel()
	}()

	err := sd.Schedule(g)
	if err != nil {
		return err
	}
	sd.Finish()

	fmt.Fprint(ChannelOut, "\r\n")

	if summary {
		cmdutils.PrintSummary(g, ChannelOut)
	}

	return nil
}

func runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t)
	if err != nil {
		return err
	}

	taskRunner.Finish()

	return nil
}

// buildTaskRunner initiates the task runner struct
//
// assigns to the global var to the package
// args are the stdin inputs of strings following the `--` parameter
//
// TODO: make this less globally and more testable
func buildTaskRunner(args []string) error {
	if err := argsValidator(args); err != nil {
		return err
	}
	vars := variables.FromMap(variableSet)
	// These are stdin args passed over `-- arg1 arg2`
	vars.Set("ArgsList", argsList)
	vars.Set("Args", strings.Join(argsList, " "))
	tr, err := runner.NewTaskRunner(runner.WithContexts(conf.Contexts), runner.WithVariables(vars), func(runner *runner.TaskRunner) {
		runner.Stdout = ChannelOut
		runner.Stderr = ChannelErr
	})

	if err != nil {
		return err
	}
	tr.OutputFormat = conf.Output
	tr.DryRun = conf.DryRun

	if conf.Quiet {
		tr.Stdout = io.Discard
		tr.Stderr = io.Discard
	}

	go func() {
		<-cancel
		tr.Cancel()
	}()

	taskRunner = tr
	return nil
}

var ErrIncorrectPipelineTaskArg = errors.New("supplied argument does not match any pipelines or tasks")

// argsValidator assigns the task or pipeline name to run
// Will have errored already if the args length is 0
//
// the first arg should be the name of the task or pipeline
func argsValidator(args []string) error {
	if conf.Pipelines[args[0]] != nil {
		pipelineName = conf.Pipelines[args[0]]
	}
	if conf.Tasks[args[0]] != nil {
		taskName = conf.Tasks[args[0]]
	}

	if pipelineName == nil && taskName == nil {
		return fmt.Errorf("%s does not exist, ensure your first argument is the name of the pipeline or task. %w", args[0], ErrIncorrectPipelineTaskArg)
	}

	argsList = args[1:]
	taskOrPipelineName = args[0]
	return nil
}
