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
var overrides []string

var tasks = make(map[string]*task.Task)
var contexts = make(map[string]*runner.ExecutionContext)
var pipelines = make(map[string]*scheduler.Pipeline)
var watchers = make(map[string]*watch.Watcher)

var cancel = make(chan struct{})
var done = make(chan bool)

func NewRootCommand(gcfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wilson",
		Short:   "Wilson the task runner",
		Version: "0.2.6",
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

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", gcfg.Debug, "enable debug")
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file to use")
	cmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "silence output")
	cmd.PersistentFlags().StringSliceVar(&overrides, "set", make([]string, 0), "override config value")

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
	gcfg, err := config.LoadGlobalConfig()
	if err != nil {
		return err
	}

	cmd := NewRootCommand(gcfg)
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
	}

	for name, def := range cfg.Contexts {
		contexts[name], err = runner.BuildContext(def, &config.Get().WilsonConfigDefinition)
		if err != nil {
			return nil, fmt.Errorf("context %s build failed: %v", name, err)
		}
	}

	for name, stages := range cfg.Pipelines {
		pipelines[name], err = scheduler.BuildPipeline(stages, cfg.Pipelines, cfg.Tasks)
		if err != nil {
			return nil, fmt.Errorf("pipeline %s build failed: %w", name, err)
		}
	}

	tr := runner.NewTaskRunner(contexts, make([]string, 0), true, false)
	for name, def := range cfg.Watchers {
		watchers[name], err = watch.BuildWatcher(name, def, tasks[def.Task], tr)
		if err != nil {
			return nil, fmt.Errorf("watcher %s build failed: %v", name, err)
		}
	}

	for k, v := range overrides {
		fmt.Println(k, v)
	}

	return cfg, nil
}
