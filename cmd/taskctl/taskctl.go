package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Version  = "0.0.0"
	Revision = "aaaa1234"
)

var (
	// Keep public for test overwrites
	// TODO: see why logrus is needed if all logging happens over
	// fmt.Fprintln()
	ChannelOut io.Writer = nil
	ChannelErr io.Writer = nil
)

var (
	debug       bool
	cfg         string
	output      string
	raw         bool
	cockpit     bool
	quiet       bool
	variableSet map[string]string
	dryRun      bool
	summary     bool
)

var TaskCtlCmd = &cobra.Command{
	Use:     "taskctl",
	Version: fmt.Sprintf("%s-%s", Version, Revision),
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
	logrus.SetOutput(ChannelErr)

	if err := TaskCtlCmd.ExecuteContext(ctx); err != nil {
		logrus.Fatal(err)
	}
}

func init() {

	TaskCtlCmd.PersistentFlags().StringVarP(&cfg, "config", "c", "tasks.yaml", "config file to use") // tasks.yaml or taskctl.yaml
	viper.BindEnv("config", "TASKCTL_CONFIG_FILE")
	viper.BindPFlag("config", TaskCtlCmd.PersistentFlags().Lookup("config"))

	TaskCtlCmd.PersistentFlags().StringVarP(&output, "output", "o", "prefixed", "output format (raw, prefixed or cockpit)")
	viper.BindEnv("output", "TASKCTL_OUTPUT_FORMAT")
	viper.BindPFlag("output", TaskCtlCmd.PersistentFlags().Lookup("output")) // TASKCTL_OUTPUT_FORMAT

	// Shortcut flags
	TaskCtlCmd.PersistentFlags().BoolVarP(&raw, "raw", "r", true, "shortcut for --output=raw")
	TaskCtlCmd.PersistentFlags().BoolVarP(&cockpit, "cockpit", "", false, "shortcut for --output=cockpit")

	// Key=Value pairs
	// can be supplied numerous times
	TaskCtlCmd.PersistentFlags().StringToStringVarP(&variableSet, "set", "", nil, "set global variable value")

	// flag toggles
	TaskCtlCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug")
	viper.BindPFlag("debug", TaskCtlCmd.PersistentFlags().Lookup("debug")) // TASKCTL_DEBUG
	TaskCtlCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "", false, "dry run")
	TaskCtlCmd.PersistentFlags().BoolVarP(&summary, "summary", "s", true, "show summary")
	TaskCtlCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quite mode")

	// Channels overwritable for
	ChannelOut = os.Stdout
	ChannelErr = os.Stderr
}

var (
	ErrIncompleteConfig = errors.New("config key is missing")
)

func initConfig() error {
	// Viper magic here
	viper.SetEnvPrefix("TASKCTL")
	viper.AutomaticEnv()

	conf = config.NewConfig()
	cl := config.NewConfigLoader(conf)

	if viper.GetString("config") == "" {
		return fmt.Errorf("config file was not provided, %w", ErrIncompleteConfig)
	}

	conf, _ = cl.Load(viper.GetString("config"))

	conf.Debug = debug
	conf.Quiet = quiet
	conf.DryRun = dryRun
	conf.Summary = summary
	conf.Output = output
	if raw {
		conf.Output = "raw"
	}
	if cockpit {
		conf.Output = "cockpit"
	}

	return nil
}
