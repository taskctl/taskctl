package main

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/watch"
	"github.com/taskctl/taskctl/pkg/context"
	"github.com/taskctl/taskctl/pkg/pipeline"
	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/task"
	"io/ioutil"
	"strings"
)

// todo: remove global variables
var debug, silent bool
var configFile string
var configValues []string

var tasks = make(map[string]*task.Task)
var contexts = make(map[string]*context.ExecutionContext)
var pipelines = make(map[string]*pipeline.Pipeline)
var watchers = make(map[string]*watch.Watcher)

var cancel = make(chan struct{})
var done = make(chan bool)
var cl config.ConfigLoader

func NewRootCommand(gcfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "taskctl",
		Short:   "Taskctl the task runner",
		Version: "0.5.1",
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

			for _, v := range configValues {
				vv := strings.Split(v, "=")
				if len(vv) == 2 {
					cl.Set(vv[0], vv[1])
				}
			}
		},
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", gcfg.Debug, "enable debug")
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file to use (taskctl.yaml or wi.yaml by default)")
	cmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "silence output")
	cmd.PersistentFlags().StringSliceVar(&configValues, "set", make([]string, 0), "override config value")

	err := cmd.MarkPersistentFlagFilename("config", "yaml", "yml")
	if err != nil {
		log.Warning(err)
	}

	cmd.AddCommand(NewListCommand())
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewWatchCommand())
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewShowCommand())

	cmd.AddCommand(NewAutocompleteCommand(cmd))

	return cmd
}

func Execute() error {
	cl = config.NewConfigLoader()
	gcfg, err := cl.LoadGlobalConfig()
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
		contexts[name], err = context.BuildContext(def, &config.Get().TaskctlConfigDefinition)
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

	tr := runner.NewTaskRunner(contexts, make([]string, 0), true, false, dryRun)
	for name, def := range cfg.Watchers {
		watchers[name], err = watch.BuildWatcher(name, def, tasks[def.Task], tr)
		if err != nil {
			return nil, fmt.Errorf("watcher %s build failed: %v", name, err)
		}
	}

	for k, v := range configValues {
		fmt.Println(k, v)
	}

	return cfg, nil
}
