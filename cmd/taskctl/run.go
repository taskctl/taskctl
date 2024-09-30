package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/scheduler"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/spf13/cobra"
)

type runCmd struct {
	cmd                    *cobra.Command
	channelOut, channelErr io.Writer
}

func newRunCmd(rootCmd *TaskCtlCmd) {
	runner := &runCmd{
		channelOut: rootCmd.ChannelOut,
		channelErr: rootCmd.ChannelErr,
	}

	rc := &cobra.Command{
		Use:     "run",
		Aliases: []string{},
		Short:   `runs <pipeline or task>`,
		Example: `taskctl run pipeline1
		taskctl run task1`,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			// display selector if nothing is supplied
			if len(args) == 0 {
				selected, err := cmdutils.DisplayTaskSelection(conf)
				if err != nil {
					return err
				}
				args = append([]string{selected}, args[0:]...)
			}

			taskRunner, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return runner.runTarget(taskRunner, conf, argsStringer)
		},
	}

	rc.AddCommand(&cobra.Command{
		Use:          "pipeline",
		Short:        `runs pipeline <task>`,
		Example:      `taskctl run pipeline pipeline:name`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			taskRunner, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return runner.runPipeline(argsStringer.pipelineName, taskRunner, conf.Summary)
		},
	})

	rc.AddCommand(&cobra.Command{
		Use:          "task",
		Aliases:      []string{},
		Short:        `runs task <task>`,
		Example:      `taskctl run task1`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			taskRunner, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return runner.runTask(argsStringer.taskName, taskRunner)
		},
	})

	runner.cmd = rc

	rootCmd.Cmd.AddCommand(runner.cmd)
}

func (r *runCmd) runTarget(taskRunner *runner.TaskRunner, conf *config.Config, argsStringer *argsToStringsMapper) (err error) {

	if argsStringer.pipelineName != nil {
		if err := r.runPipeline(argsStringer.pipelineName, taskRunner, conf.Summary); err != nil {
			return fmt.Errorf("pipeline %s failed: %w", argsStringer.taskOrPipelineName, err)
		}
		return nil
	}

	if argsStringer.taskName != nil {
		if err := r.runTask(argsStringer.taskName, taskRunner); err != nil {
			return fmt.Errorf("task %s failed: %w", argsStringer.taskOrPipelineName, err)
		}
	}

	return nil
}

func (r *runCmd) runPipeline(g *scheduler.ExecutionGraph, taskRunner *runner.TaskRunner, summary bool) error {
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

	fmt.Fprint(r.channelOut, "\r\n")

	if summary {
		cmdutils.PrintSummary(g, r.channelOut)
	}

	return nil
}

func (r *runCmd) runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t)
	if err != nil {
		return err
	}

	taskRunner.Finish()

	return nil
}

var ErrIncorrectPipelineTaskArg = errors.New("supplied argument does not match any pipelines or tasks")

// // Arg munging
// var (
// 	taskOrPipelineName string                    = ""
// 	pipelineName       *scheduler.ExecutionGraph = nil
// 	taskName           *task.Task                = nil
// 	argsList           []string                  = nil
// )

type argsToStringsMapper struct {
	taskOrPipelineName string
	pipelineName       *scheduler.ExecutionGraph
	taskName           *task.Task
	argsList           []string
}

// argsValidator assigns the task or pipeline name to run
// Will have errored already if the args length is 0
//
// the first arg should be the name of the task or pipeline
func argsValidator(args []string, conf *config.Config) (*argsToStringsMapper, error) {
	argsStringer := &argsToStringsMapper{}

	if conf.Pipelines[args[0]] != nil {
		argsStringer.pipelineName = conf.Pipelines[args[0]]
	}
	if conf.Tasks[args[0]] != nil {
		argsStringer.taskName = conf.Tasks[args[0]]
	}

	if argsStringer.pipelineName == nil && argsStringer.taskName == nil && conf.Watchers[args[0]] == nil {
		return argsStringer, fmt.Errorf("%s does not exist, ensure your first argument is the name of the pipeline or task. %w", args[0], ErrIncorrectPipelineTaskArg)
	}

	argsStringer.argsList = args[1:]
	argsStringer.taskOrPipelineName = args[0]
	return argsStringer, nil
}
