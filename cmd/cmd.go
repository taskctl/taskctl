package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/task"
	"io/ioutil"
	"os"
)

var debug, silent bool
var configFile string
var cfg *config.Config

var tasks = make(map[string]*task.Task)
var contexts = make(map[string]*runner.Context)
var pipelines = make(map[string]*task.Pipeline)

var cancel = make(chan struct{})
var done = make(chan bool)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Short:   "Wilson the task runner",
		Version: "0.1.0-beta",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if silent {
				logrus.SetOutput(ioutil.Discard)
				quiet = true
			}

			var err error
			cfg, err = config.Load(configFile)
			if err != nil {
				logrus.Fatal(err)
			}

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
				pipelines[name] = task.BuildPipeline(def.Tasks, tasks)
			}
		},
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug")
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "wilson.yaml", "config file to use")
	cmd.PersistentFlags().BoolVarP(&quiet, "silent", "q", false, "silence output")

	cmd.AddCommand(NewWatchCommand())
	cmd.AddCommand(NewRunCommand())
	cmd.AddCommand(NewCompletionsCommand(cmd))

	return cmd
}

func NewCompletionsCommand(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion",
		Short: "Generates bash completion scripts",
		Long: `To load completion run

. <(bitbucket completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(bitbucket completion)
`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout)
		},
	}
}

func Execute() error {
	cmd := NewRootCommand()
	return cmd.Execute()
}

func Abort() {
	close(cancel)
	<-done
}
