package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/Ensono/taskctl/internal/config"
	outputpkg "github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/variables"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	Version  = "0.0.0"
	Revision = "aaaa1234"
)

var cancel = make(chan struct{})

type rootCmdFlags struct {
	// all vars here
	Debug       bool
	CfgFilePath string
	Output      string
	Raw         bool
	Cockpit     bool
	Quiet       bool
	VariableSet map[string]string
	DryRun      bool
	// Summary report at the end of the task run
	// this was set to default as true in the original
	// - not sure this makes sense for a boolean flag "¯\_(ツ)_/¯"
	Summary bool
}

type TaskCtlCmd struct {
	Cmd        *cobra.Command
	ChannelOut io.Writer
	ChannelErr io.Writer
	viperConf  *viper.Viper
	rootFlags  *rootCmdFlags
}

func NewTaskCtlCmd(channelOut, channelErr io.Writer) *TaskCtlCmd {
	tc := &TaskCtlCmd{
		ChannelOut: channelOut,
		ChannelErr: channelErr,
		Cmd: &cobra.Command{
			Use:     "taskctl",
			Version: fmt.Sprintf("%s-%s", Version, Revision),
			Args:    cobra.ExactArgs(0),
			Short:   "modern task runner",
			Long: `Concurrent task runner, developer's routine tasks automation toolkit.
			Simple modern alternative to GNU Make 🧰`, // taken from original GH repo
			SuggestionsMinimumDistance: 1,
		},
	}

	tc.rootFlags = &rootCmdFlags{}
	tc.viperConf = viper.NewWithOptions()
	tc.viperConf.SetEnvPrefix("TASKCTL")
	tc.viperConf.SetConfigName("taskctl_conf")

	tc.Cmd.PersistentFlags().StringVarP(&tc.rootFlags.CfgFilePath, "config", "c", "taskctl.yaml", "config file to use") // tasks.yaml or taskctl.yaml
	if err := tc.viperConf.BindEnv("config", "TASKCTL_CONFIG_FILE"); err != nil {
		log.Fatal(err)
	}
	if err := tc.viperConf.BindPFlag("config", tc.Cmd.PersistentFlags().Lookup("config")); err != nil {
		log.Fatal(err)
	}

	tc.Cmd.PersistentFlags().StringVarP(&tc.rootFlags.Output, "output", "o", string(outputpkg.PrefixedOutput), "output format (raw, prefixed or cockpit)")
	_ = tc.viperConf.BindEnv("output", "TASKCTL_OUTPUT_FORMAT")
	_ = tc.viperConf.BindPFlag("output", tc.Cmd.PersistentFlags().Lookup("output"))

	// Shortcut flags
	tc.Cmd.PersistentFlags().BoolVarP(&tc.rootFlags.Raw, "raw", "r", false, "shortcut for --output=raw")
	_ = tc.viperConf.BindPFlag("raw", tc.Cmd.PersistentFlags().Lookup("raw")) // TASKCTL_DEBUG

	tc.Cmd.PersistentFlags().BoolVarP(&tc.rootFlags.Cockpit, "cockpit", "", false, "shortcut for --output=cockpit")
	_ = tc.viperConf.BindPFlag("cockpit", tc.Cmd.PersistentFlags().Lookup("cockpit")) // TASKCTL_DEBUG

	// Key=Value pairs
	// can be supplied numerous times
	tc.Cmd.PersistentFlags().StringToStringVarP(&tc.rootFlags.VariableSet, "set", "", nil, "set global variable value")
	_ = tc.viperConf.BindPFlag("set", tc.Cmd.PersistentFlags().Lookup("set")) // TASKCTL_DEBUG

	// flag toggles
	tc.Cmd.PersistentFlags().BoolVarP(&tc.rootFlags.Debug, "debug", "d", false, "enable debug")
	_ = tc.viperConf.BindPFlag("debug", tc.Cmd.PersistentFlags().Lookup("debug")) // TASKCTL_DEBUG

	tc.Cmd.PersistentFlags().BoolVarP(&tc.rootFlags.DryRun, "dry-run", "", false, "dry run")
	_ = tc.viperConf.BindPFlag("dry-run", tc.Cmd.PersistentFlags().Lookup("dry-run"))

	tc.Cmd.PersistentFlags().BoolVarP(&tc.rootFlags.Summary, "summary", "s", true, "show summary")
	_ = tc.viperConf.BindPFlag("summary", tc.Cmd.PersistentFlags().Lookup("summary"))

	tc.Cmd.PersistentFlags().BoolVarP(&tc.rootFlags.Quiet, "quiet", "q", false, "quite mode")
	_ = tc.viperConf.BindPFlag("quiet", tc.Cmd.PersistentFlags().Lookup("quiet"))

	return tc
}

func (tc *TaskCtlCmd) InitCommand() error {
	// add all sub commands
	// TODO: perhaps think about a better way of doing this
	newRunCmd(tc)
	newGraphCmd(tc)
	newShowCmd(tc)
	newListCmd(tc)
	newInitCmd(tc)
	newValidateCmd(tc)
	newWatchCmd(tc)
	return nil
}

func (tc *TaskCtlCmd) Execute(ctx context.Context) error {
	// NOTE: do we need logrus ???
	// latest Go has structured logging...
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:   false,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   false,
	})
	logrus.SetOutput(tc.ChannelErr)

	return tc.Cmd.ExecuteContext(ctx)
}

var (
	ErrIncompleteConfig = errors.New("config key is missing")
)

func (tc *TaskCtlCmd) initConfig() (*config.Config, error) {
	// Viper magic here
	tc.viperConf.AutomaticEnv()
	configFilePath := tc.viperConf.GetString("config")
	if _, err := os.Stat(configFilePath); err != nil {
		return nil, fmt.Errorf("%w\nincorrect config file (%s) does not exist", ErrIncompleteConfig, configFilePath)
	}

	cl := config.NewConfigLoader(config.NewConfig())

	conf, err := cl.Load(configFilePath)
	if err != nil {
		return nil, err
	}
	// if cmd line flags were passed in, they override anything
	// parsed from theconfig file
	if !conf.Debug {
		conf.Debug = tc.viperConf.GetBool("debug") // this is bound to viper env flag
	}

	if conf.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if !conf.Quiet {
		conf.Quiet = tc.rootFlags.Quiet
	}
	if !conf.DryRun {
		conf.DryRun = tc.rootFlags.Quiet
	}
	if !conf.Summary {
		conf.Summary = tc.rootFlags.Summary
	}

	if conf.Output == "" {
		conf.Output = outputpkg.OutputEnum(tc.viperConf.GetString("output"))
	}

	// if cmdline flags for output shorthand has been provided we
	// overwrite the output
	if tc.viperConf.GetBool("raw") {
		conf.Output = outputpkg.RawOutput
	}

	if tc.viperConf.GetBool("cockpit") {
		conf.Output = outputpkg.CockpitOutput
	}

	// these are CLI args only
	conf.Options.GraphOrientationLeftRight = tc.viperConf.GetBool("lr")
	conf.Options.InitDir = tc.viperConf.GetString("dir")
	conf.Options.InitNoPrompt = tc.viperConf.GetBool("no-prompt")
	return conf, nil
}

// buildTaskRunner initiates the task runner struct
//
// assigns to the global var to the package
// args are the stdin inputs of strings following the `--` parameter
func (tc *TaskCtlCmd) buildTaskRunner(args []string, conf *config.Config) (*runner.TaskRunner, *argsToStringsMapper, error) {
	argsStringer, err := argsValidator(args, conf)
	if err != nil {
		return nil, nil, err
	}
	// fmt.Println(viper.GetStringMapString("set"))
	vars := variables.FromMap(tc.viperConf.GetStringMapString("set"))
	// These are stdin args passed over `-- arg1 arg2`
	vars.Set("ArgsList", argsStringer.argsList)
	vars.Set("Args", strings.Join(argsStringer.argsList, " "))
	tr, err := runner.NewTaskRunner(runner.WithContexts(conf.Contexts), runner.WithVariables(vars), func(runner *runner.TaskRunner) {
		runner.Stdout = tc.ChannelOut
		runner.Stderr = tc.ChannelErr
	})

	if err != nil {
		return nil, nil, err
	}
	tr.OutputFormat = string(conf.Output)
	tr.DryRun = conf.DryRun

	if conf.Quiet {
		tr.Stdout = io.Discard
		tr.Stderr = io.Discard
	}

	go func() {
		<-cancel
		tr.Cancel()
	}()

	return tr, argsStringer, nil
}
