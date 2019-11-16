package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/scheduler"
	"github.com/trntv/wilson/pkg/task"
	"io/ioutil"
	"os"
	"strings"
)

var debug, silent bool
var cfg *config.Config

var configFile string

var tasks = make(map[string]*task.Task)
var contexts = make(map[string]*runner.Context)
var pipelines = make(map[string]*scheduler.Pipeline)

var cancel = make(chan struct{})
var done = make(chan bool)

func NewRootCommand() *cobra.Command {
	loadConfig()
	cmd := &cobra.Command{
		Short:   "Wilson the task runner",
		Version: "0.1.0-beta.2",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(log.DebugLevel)
			} else {
				log.SetLevel(log.WarnLevel)
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

	cmd.AddCommand(NewWatchCommand())
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewCompletionsCommand(cmd))

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

func parseConfigFlag() string {
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--config") {
			file := strings.TrimPrefix(arg, "--config")
			file = strings.TrimLeft(file, " =")
			if file != "" {
				return file
			}

			if len(os.Args) >= i+2 {
				return os.Args[i+1]
			}
		}
	}

	return ""
}

func loadConfig() {
	var err error
	configFile = parseConfigFlag()
	cfg, err = config.Load(configFile)
	if err != nil {
		log.Fatal(err)
	}

	for name, def := range cfg.Tasks {
		tasks[name] = task.BuildTask(def)
		tasks[name].Name = name
	}

	for name, def := range cfg.Contexts {
		contexts[name], err = runner.BuildContext(def)
		if err != nil {
			log.Fatalf("context %s build failed: %v", name, err)
		}
	}

	for name, def := range cfg.Pipelines {
		pipelines[name] = scheduler.BuildPipeline(def.Tasks, tasks)
	}
}

func mapNames(m interface{}) (list []string) {
	var mm map[string]interface{}
	mm, ok := m.(map[string]interface{})
	if !ok {
		return list
	}

	var i int
	list = make([]string, len(mm))

	for name := range mm {
		list[i] = name
		i++
	}

	return list
}
