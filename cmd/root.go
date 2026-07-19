package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/tui"
	"github.com/taskctl/taskctl/runner"
)

const (
	groupRun     = "run"
	groupInspect = "inspect"
	groupSetup   = "setup"
)

var stdin io.ReadCloser = os.Stdin

// Stdin returns the reader used for interactive prompts.
func Stdin() io.ReadCloser { return stdin }

// SetStdin overrides the reader used for interactive prompts (test seam).
func SetStdin(newStdin io.ReadCloser) { stdin = newStdin }

// Run builds the CLI and executes it, cancelling the run context on SIGINT or
// SIGTERM so in-flight tasks tear down gracefully.
func Run(version string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := NewRootCommand(version)
	root.SetContext(ctx)

	// ExecuteC returns the command that actually failed, which present needs to
	// print the right usage for arg/flag errors.
	cmd, err := root.ExecuteC()
	if code := present(cmd, err); code != 0 {
		return exitError{code}
	}
	return nil
}

// NewRootCommand builds the taskctl root command with all subcommands. The
// config and loader are created once here and shared with every subcommand;
// PersistentPreRunE populates the config in place before any RunE fires.
func NewRootCommand(version string) *cobra.Command {
	cfg := config.NewConfig()
	loader := config.NewConfigLoader(cfg)
	cl := &loader

	root := &cobra.Command{
		Use:           "taskctl [target...] [-- task-args]",
		Short:         "modern task runner",
		Version:       version,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: "  taskctl prepare\n" +
			"  taskctl list --output json\n" +
			"  taskctl run test -- -v",
		ValidArgsFunction: targetCompletion(cfg),
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return resolveConfig(cmd, cfg, cl)
		},
		// A bare invocation runs the given targets, or opens the interactive
		// selector when none are given and prompts are possible.
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, _ := splitArgsAtDash(cmd, args)
			if len(targets) > 0 {
				return runTargets(cmd, cfg, targets, false)
			}

			// No target: skip the selector when prompts are suppressed
			// (--no-input/json) or stdin isn't a real terminal (the selector
			// needs a TTY), rather than blocking on or silently running it.
			if nonInteractive(cmd, cfg) || !tui.Interactive(stdin) {
				return errors.New("no target specified; run 'taskctl list' to see available targets")
			}

			suggestions := buildSuggestions(cfg)
			if len(suggestions) == 0 {
				return errors.New("no tasks or pipelines found in config")
			}

			items := make([]tui.Item[suggestion], 0, len(suggestions))
			for _, s := range suggestions {
				items = append(items, tui.Item[suggestion]{Label: s.DisplayName, Value: s})
			}

			selection, err := tui.Select(stdin, "Select task to run", items)
			if err != nil {
				if errors.Is(err, tui.ErrAborted) {
					return nil
				}
				return err
			}

			return runTargets(cmd, cfg, []string{selection.Target}, false)
		},
	}

	fs := root.PersistentFlags()
	fs.BoolP("debug", "d", false, "enable debug")
	fs.StringP("config", "c", "", "config file to use (default tasks.yaml or taskctl.yaml)")
	fs.StringP("output", "o", "", "output format (default, prefixed, raw or json)")
	fs.BoolP("raw", "r", false, "shortcut for --output=raw")
	fs.BoolP("quiet", "q", false, "quiet mode")
	fs.StringSlice("set", nil, "set global variable value")
	fs.Bool("dry-run", false, "dry run")
	fs.BoolP("summary", "s", true, "show summary")
	fs.Bool("no-input", false, "disable interactive prompts")

	_ = root.RegisterFlagCompletionFunc("output", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{output.FormatDefault, output.FormatPrefixed, output.FormatRaw, output.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})

	root.SetFlagErrorFunc(func(_ *cobra.Command, e error) error { return usageError{e} })

	root.AddGroup(
		&cobra.Group{ID: groupRun, Title: "Run:"},
		&cobra.Group{ID: groupInspect, Title: "Inspect:"},
		&cobra.Group{ID: groupSetup, Title: "Setup:"},
	)
	root.SetHelpCommandGroupID(groupSetup)
	root.SetCompletionCommandGroupID(groupSetup)

	root.AddCommand(
		newRunCommand(cfg),
		newInitCommand(cfg),
		newListCommand(cfg),
		newShowCommand(cfg),
		newWatchCommand(cfg),
		newGraphCommand(cfg),
		newValidateCommand(cfg),
		newSkillCommand(),
	)

	markUsageErrors(root)

	return root
}

// resolveConfig loads the config file and resolves the effective output format,
// log level, quiet and dry-run settings into cfg. It tolerates a missing
// default config file (only an explicit --config that is missing is fatal) so
// commands like init and validate work without one.
func resolveConfig(cmd *cobra.Command, cfg *config.Config, cl *config.Loader) error {
	fs := cmd.Flags()
	for _, b := range []struct{ name, env string }{
		{"debug", "TASKCTL_DEBUG"},
		{"config", "TASKCTL_CONFIG_FILE"},
		{"no-input", "TASKCTL_NO_INPUT"},
	} {
		if err := bindEnv(fs, b.name, b.env); err != nil {
			return err
		}
	}

	configFile, _ := fs.GetString("config")
	if _, err := cl.Load(configFile); err != nil {
		if !errors.Is(err, config.ErrConfigNotFound) || fs.Changed("config") {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	set, _ := fs.GetStringSlice("set")
	for _, kv := range set {
		if k, v, ok := strings.Cut(kv, "="); ok {
			cfg.Variables.Set(k, v)
		}
	}

	// An explicit --debug/TASKCTL_DEBUG wins in both directions; the config
	// file's debug: only applies when the flag was not set, so --debug=false
	// can turn off a config that enables it.
	debug, _ := fs.GetBool("debug")
	if !fs.Changed("debug") {
		debug = cfg.Debug
	}
	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	if quiet, _ := fs.GetBool("quiet"); quiet {
		cmd.Root().SetErr(io.Discard)
		slog.SetDefault(slog.New(slog.DiscardHandler))
		cfg.Quiet = true
	}

	// Precedence: an explicit --output wins, then --raw, then the
	// TASKCTL_OUTPUT_FORMAT env var, then a config-file output:, then the
	// built-in default. The env var is read here rather than bound onto the
	// --output flag so a command-line --raw stays authoritative over it.
	raw, _ := fs.GetBool("raw")
	switch {
	case fs.Changed("output"):
		cfg.Output, _ = fs.GetString("output")
	case raw:
		cfg.Output = output.FormatRaw
	case os.Getenv("TASKCTL_OUTPUT_FORMAT") != "":
		cfg.Output = os.Getenv("TASKCTL_OUTPUT_FORMAT")
	case cfg.Output == "":
		cfg.Output = output.FormatDefault
	}

	if cfg.Output == output.FormatDefault && !tui.Interactive(os.Stdout) {
		cfg.Output = output.FormatPrefixed
	}

	switch cfg.Output {
	case output.FormatRaw, output.FormatPrefixed, output.FormatDefault, output.FormatJSON:
	default:
		return fmt.Errorf("unknown output format %q (want raw, prefixed, default or json)", cfg.Output)
	}

	if dryRun, _ := fs.GetBool("dry-run"); dryRun {
		cfg.DryRun = true
	}

	return nil
}

// bindEnv applies an environment variable to a flag that was not set on the
// command line, returning an error when the env value is invalid for the flag
// (e.g. a non-boolean TASKCTL_DEBUG). pflag has no built-in env support, so
// this fills the gap.
func bindEnv(fs *pflag.FlagSet, name, env string) error {
	if fs.Changed(name) {
		return nil
	}
	v, ok := os.LookupEnv(env)
	if !ok {
		return nil
	}
	if err := fs.Set(name, v); err != nil {
		return fmt.Errorf("invalid value for %s: %w", env, err)
	}
	return nil
}

// nonInteractive reports whether the CLI should skip prompts entirely and rely
// on defaults: when explicitly requested via --no-input/TASKCTL_NO_INPUT, or
// when producing machine-readable JSON output. A non-TTY stdin alone does not
// count — huh still prompts in accessible (line-based) mode against a pipe.
func nonInteractive(cmd *cobra.Command, cfg *config.Config) bool {
	noInput, _ := cmd.Flags().GetBool("no-input")
	return noInput || cfg.Output == output.FormatJSON
}

func buildTaskRunner(cmd *cobra.Command, cfg *config.Config) (*runner.TaskRunner, error) {
	_, passArgs := splitArgsAtDash(cmd, cmd.Flags().Args())
	variables := cfg.Variables.With("Args", strings.Join(passArgs, " "))
	variables.Set("ArgsList", passArgs)

	taskRunner, err := runner.NewTaskRunner(runner.WithContexts(cfg.Contexts), runner.WithVariables(variables))
	if err != nil {
		return nil, err
	}

	taskRunner.OutputFormat = cfg.Output
	taskRunner.DryRun = cfg.DryRun

	if cfg.Quiet {
		taskRunner.Stdout = io.Discard
		taskRunner.Stderr = io.Discard
	}

	ctx := cmd.Context()
	go func() {
		<-ctx.Done()
		taskRunner.Cancel()
	}()

	return taskRunner, nil
}

type suggestion struct {
	Target, DisplayName string
	IsTask              bool
}

func buildSuggestions(cfg *config.Config) []suggestion {
	if cfg == nil {
		return nil
	}

	suggestions := make([]suggestion, 0)

	for _, v := range slices.Collect(maps.Keys(cfg.Pipelines)) {
		suggestions = append(suggestions, suggestion{
			Target:      v,
			DisplayName: v + tui.StyleFaint.Render(" - pipeline"),
		})
	}

	for k, v := range cfg.Tasks {
		desc := "task"
		if v.Description != "" {
			desc = v.Description
		}
		suggestions = append(suggestions, suggestion{
			Target:      k,
			DisplayName: k + tui.StyleFaint.Render(" - "+desc),
			IsTask:      true,
		})
	}

	slices.SortFunc(suggestions, func(a, b suggestion) int {
		return strings.Compare(a.Target, b.Target)
	})

	return suggestions
}

// completionFunc returns a shell-completion function that loads the config
// itself and offers whatever names produces. Completion runs before
// PersistentPreRunE (so bindEnv has not fired yet); it reads the --config flag
// and falls back to TASKCTL_CONFIG_FILE directly to keep completion working
// for users who configure via the environment.
func completionFunc(cfg *config.Config, names func() []string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		loader := config.NewConfigLoader(cfg)
		configFile, _ := cmd.Flags().GetString("config")
		if configFile == "" {
			configFile = os.Getenv("TASKCTL_CONFIG_FILE")
		}
		if _, err := loader.Load(configFile); err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return names(), cobra.ShellCompDirectiveNoFileComp
	}
}

func targetCompletion(cfg *config.Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completionFunc(cfg, func() []string {
		names := make([]string, 0)
		for _, s := range buildSuggestions(cfg) {
			names = append(names, s.Target)
		}
		return names
	})
}
