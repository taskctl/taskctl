package cmd

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var BuildVersion = "dev"

// var stdin io.ReadCloser

// var cancel = make(chan struct{})

// var cfg *config.Config

var (
	debug   bool
	config  string
	output  string
	raw     bool
	cockpit bool
	quiet   bool
	set     map[string]string

	dryRun  bool
	summary bool
)

var TaskCtlCmd = &cobra.Command{
	Use:     "taskctl",
	Version: BuildVersion,
	Short:   "modern task runner",
	Long:    `Concurrent task runner, developer's routine tasks automation toolkit. Simple modern alternative to GNU Make 🧰`, // taken from original GH repo
}

func Execute(ctx context.Context) {
	// NOTE: do we need logrus ???
	// latest Go has structured logging...
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   false,
	})

	if err := TaskCtlCmd.ExecuteContext(ctx); err != nil {
		logrus.Fatal(err)
	}
}

func init() {

	TaskCtlCmd.PersistentFlags().StringVarP(&config, "config", "c", "tasks.yaml", "config file to use") // tasks.yaml or taskctl.yaml
	viper.BindPFlag("config", TaskCtlCmd.PersistentFlags().Lookup("config"))                            // TASKCTL_CONFIG_FILE

	TaskCtlCmd.PersistentFlags().StringVarP(&output, "output", "c", "raw", "output format (raw, prefixed or cockpit)")
	viper.BindPFlag("output", TaskCtlCmd.PersistentFlags().Lookup("output")) // TASKCTL_OUTPUT_FORMAT

	// Shortcut flags
	TaskCtlCmd.PersistentFlags().BoolVarP(&raw, "raw", "r", true, "shortcut for --output=raw")
	TaskCtlCmd.PersistentFlags().BoolVarP(&cockpit, "cockpit", "", false, "shortcut for --output=cockpit")

	// Key=Value pairs
	// can be supplied numerous times
	TaskCtlCmd.PersistentFlags().StringToStringVarP(&set, "set", "", nil, "set global variable value")

	// flag toggles
	TaskCtlCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug")
	viper.BindPFlag("debug", TaskCtlCmd.PersistentFlags().Lookup("debug")) // TASKCTL_DEBUG
	TaskCtlCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "", false, "dry run")
	TaskCtlCmd.PersistentFlags().BoolVarP(&summary, "summary", "s", true, "show summary")
	TaskCtlCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quite mode")
}
