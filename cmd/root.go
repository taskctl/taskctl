package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/task"
)

var debug bool
var configFile string

var tasks map[string]*task.Task
var contexts map[string]*runner.Context
var pipelines map[string]*task.Pipeline

var cancel = make(chan struct{})
var done = make(chan bool)

var rootCmd = &cobra.Command{
	Short: "Wilson the task runner",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// todo: better solution
		if cmd.Use == "help [command]" {
			return
		}

		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		cfg, err := config.Load(configFile)
		if err != nil {
			logrus.Fatalln(err)
		}

		tasks = make(map[string]*task.Task)
		contexts = make(map[string]*runner.Context)
		pipelines = make(map[string]*task.Pipeline)

		for name, def := range cfg.Tasks {
			tasks[name] = task.BuildTask(def)
			tasks[name].Name = name
		}

		for name, def := range cfg.Contexts {
			contexts[name], err = runner.BuildContext(def)
			if err != nil {
				logrus.Fatalf("context %s build failed: %v", name, err)
			}
		}

		for name, def := range cfg.Pipelines {
			pipelines[name] = task.BuildPipeline(def, tasks)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "wilson.yaml", "config file to use")
}

func Execute() error {
	return rootCmd.Execute()
}

func Abort() {
	close(cancel)
	<-done
}
