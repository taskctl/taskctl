package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/internal/config"
	"github.com/trntv/wilson/internal/watch"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/scheduler"
	"github.com/trntv/wilson/pkg/task"
	"io/ioutil"
)

// todo: remove global variables
var debug, silent bool
var configFile string

var tasks = make(map[string]*task.Task)
var contexts = make(map[string]*runner.ExecutionContext)
var pipelines = make(map[string]*scheduler.Pipeline)
var watchers = make(map[string]*watch.Watcher)

var cancel = make(chan struct{})
var done = make(chan bool)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wilson",
		Short:   "Wilson the task runner",
		Version: "0.2.1",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(log.DebugLevel)
			} else {
				log.SetLevel(log.InfoLevel)
			}

			if silent {
				log.SetOutput(ioutil.Discard)
				quiet = true
			}
		},
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug")
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file to use")
	cmd.PersistentFlags().BoolVarP(&quiet, "silent", "q", false, "silence output")

	err := cmd.MarkPersistentFlagFilename("config", "yaml", "yml")
	if err != nil {
		log.Warning(err)
	}

	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewWatchCommand())
	cmd.AddCommand(NewInitCommand())

	cmd.AddCommand(NewAutocompleteCommand(cmd))

	return cmd
}

func Execute() error {
	cmd := NewRootCommand()
	return cmd.Execute()
}

func Abort() {
	close(cancel)
	<-done
}

func loadConfig() (cfg *config.Config, err error) {
	cfg, err = config.Load(configFile)
	if err != nil {
		return nil, err
	}

	for name, def := range cfg.Tasks {
		tasks[name] = task.BuildTask(def)
		tasks[name].Name = name
	}

	for name, def := range cfg.Contexts {
		contexts[name], err = runner.BuildContext(def, &config.Get().WilsonConfig)
		if err != nil {
			return nil, fmt.Errorf("context %s build failed: %v", name, err)
		}
	}

	for name, stages := range cfg.Pipelines {
		pipelines[name], err = scheduler.BuildPipeline(stages, tasks)
		if err != nil {
			return nil, err
		}
	}

	tr := runner.NewTaskRunner(contexts, make([]string, 0), true, false)
	for name, def := range cfg.Watchers {
		watchers[name], err = watch.BuildWatcher(name, def, tasks[def.Task], tr)
		if err != nil {
			return nil, fmt.Errorf("watcher %s build failed: %v", name, err)
		}
	}

	return cfg, nil
}
