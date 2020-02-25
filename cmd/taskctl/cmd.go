package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/taskctl/taskctl/pkg/output"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/watch"
	"github.com/taskctl/taskctl/pkg/context"
	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/task"
)

var taskctlCmd *cobra.Command

var debug, quiet bool
var configFile, oflavor string
var configValues []string

var tasks = make(map[string]*task.Task)
var contexts = make(map[string]*context.ExecutionContext)
var pipelines = make(map[string]*pipeline.Pipeline)
var watchers = make(map[string]*watch.Watcher)

var cancel = make(chan struct{})
var done = make(chan bool)
var cl config.ConfigLoader

func NewRootCommand() *cobra.Command {
	cfg := config.Get()
	taskctlCmd = &cobra.Command{
		Use:               "taskctl",
		Short:             "Taskctl the task runner",
		Version:           "0.6.1",
		DisableAutoGenTag: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logrus.SetFormatter(&logrus.TextFormatter{
				DisableColors:   false,
				TimestampFormat: "2006-01-02 15:04:05",
				FullTimestamp:   false,
			})

			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			} else {
				logrus.SetLevel(logrus.InfoLevel)
			}

			if quiet {
				logrus.SetOutput(ioutil.Discard)
				output.SetStdout(ioutil.Discard)
			}

			for _, v := range configValues {
				vv := strings.Split(v, "=")
				if len(vv) == 2 {
					cl.Set(vv[0], vv[1])
				}
			}
		},
	}

	taskctlCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", cfg.Debug, "enable debug")
	taskctlCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file to use (tasks.yaml or taskctl.yaml by default)")
	taskctlCmd.PersistentFlags().StringVarP(&oflavor, "output", "o", cfg.Output, "output flavour")
	taskctlCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quite mode")
	taskctlCmd.PersistentFlags().StringSliceVar(&configValues, "set", make([]string, 0), "override config value")

	taskctlCmd.AddCommand(NewListCommand())
	taskctlCmd.AddCommand(NewRunCommand())
	taskctlCmd.AddCommand(NewWatchCommand())
	taskctlCmd.AddCommand(NewInitCommand())
	taskctlCmd.AddCommand(NewShowCommand())

	taskctlCmd.AddCommand(NewAutocompleteCommand(taskctlCmd))

	return taskctlCmd
}

func Execute() error {
	cl = config.NewConfigLoader()
	_, err := cl.LoadGlobalConfig()
	if err != nil {
		return err
	}

	cmd := NewRootCommand()
	return cmd.Execute()
}

func Abort() {
	close(cancel)
	<-done
}

func loadConfig() (cfg *config.Config, err error) {
	cfg, err = cl.Load(configFile)
	if err != nil {
		if configFile != "" || !errors.Is(err, config.ErrConfigNotFound) {
			return nil, err
		}
	}

	for name, def := range cfg.Tasks {
		tasks[name] = task.BuildTask(def)
	}

	for name, def := range cfg.Contexts {
		contexts[name], err = context.BuildContext(def, config.Get())
		if err != nil {
			return nil, fmt.Errorf("context %s build failed: %v", name, err)
		}
	}

	for name, stages := range cfg.Pipelines {
		pipelines[name], err = pipeline.BuildPipeline(stages, cfg.Pipelines, cfg.Tasks)
		if err != nil {
			return nil, fmt.Errorf("pipeline %s build failed: %w", name, err)
		}
	}

	for name, def := range cfg.Watchers {
		watchers[name], err = watch.BuildWatcher(name, def, tasks[def.Task])
		if err != nil {
			return nil, fmt.Errorf("watcher %s build failed: %v", name, err)
		}
	}

	for k, v := range configValues {
		fmt.Println(k, v)
	}

	return cfg, nil
}
