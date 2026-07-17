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
	"sync"
	"syscall"

	"github.com/taskctl/taskctl/runner"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/output"
)

var stdin io.ReadCloser
var cancelFn func()
var cancelMu sync.Mutex
var cfg *config.Config
var au aurora.Aurora = aurora.NewAurora(false)

func isTTY(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

func defaultStdinIsTTY() bool  { return isTTY(os.Stdin.Fd()) }
func defaultStdoutIsTTY() bool { return isTTY(os.Stdout.Fd()) }

// stdinIsTTY / stdoutIsTTY report whether stdin / stdout are interactive
// terminals. They are package-level vars so tests can stub them without a real TTY.
var stdinIsTTY = defaultStdinIsTTY
var stdoutIsTTY = defaultStdoutIsTTY

func Stdin() io.ReadCloser {
	return stdin
}

func SetStdin(newStdin io.ReadCloser) {
	stdin = newStdin
}

// SetStdinIsTTY overrides the stdin TTY detection function; intended for
// tests. Pass nil to restore the default isatty-based detection.
func SetStdinIsTTY(f func() bool) {
	if f == nil {
		stdinIsTTY = defaultStdinIsTTY
		return
	}
	stdinIsTTY = f
}

// SetStdoutIsTTY overrides the stdout TTY detection function; intended for
// tests. Pass nil to restore the default isatty-based detection.
func SetStdoutIsTTY(f func() bool) {
	if f == nil {
		stdoutIsTTY = defaultStdoutIsTTY
		return
	}
	stdoutIsTTY = f
}

// nonInteractive reports whether the CLI should avoid interactive prompts:
// when explicitly requested via --no-input/TASKCTL_NO_INPUT, when producing
// machine-readable JSON output, or when stdin is not a terminal.
func nonInteractive(c *cli.Context) bool {
	return c.Bool("no-input") || cfg.Output == output.FormatJSON || !stdinIsTTY()
}

func Run(version string) error {
	stdin = os.Stdin
	app := NewApp(version)

	go listenSignals()

	return app.Run(os.Args)
}

func NewApp(version string) *cli.App {
	cfg = config.NewConfig()
	cl := config.NewConfigLoader(cfg)

	return &cli.App{
		Name:                 "taskctl",
		Usage:                "modern task runner",
		Version:              version,
		EnableBashCompletion: true,
		BashComplete: func(c *cli.Context) {
			cfg, _ = cl.Load(c.String("config"))
			suggestions := buildSuggestions(cfg)

			for _, v := range suggestions {
				candidate := strings.ReplaceAll(v.Target, ":", "\\:")
				fmt.Printf("%s\n", candidate)
			}

			cli.DefaultAppComplete(c)
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "enable debug",
				EnvVars: []string{"TASKCTL_DEBUG"},
			},
			&cli.PathFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "config file to use",
				EnvVars:     []string{"TASKCTL_CONFIG_FILE"},
				DefaultText: "tasks.yaml or taskctl.yaml",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output format (raw, prefixed, cockpit or json)",
				EnvVars: []string{"TASKCTL_OUTPUT_FORMAT"},
			},
			&cli.BoolFlag{
				Name:    "raw",
				Aliases: []string{"r"},
				Usage:   "shortcut for --output=raw",
			},
			&cli.BoolFlag{
				Name:  "cockpit",
				Usage: "shortcut for --output=cockpit",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "quite mode",
			},
			&cli.StringSliceFlag{
				Name:  "set",
				Usage: "set global variable value",
			},
			&cli.BoolFlag{
				Name:  "dry-Run",
				Usage: "dry Run",
			},
			&cli.BoolFlag{
				Name:    "summary",
				Usage:   "show summary",
				Aliases: []string{"s"},
				Value:   true,
			},
			&cli.BoolFlag{
				Name:    "no-input",
				Usage:   "disable interactive prompts",
				EnvVars: []string{"TASKCTL_NO_INPUT"},
			},
		},
		Before: func(c *cli.Context) (err error) {
			cfg, err = cl.Load(c.String("config"))
			if err != nil && (c.IsSet("config") && errors.Is(err, config.ErrConfigNotFound) || !errors.Is(err, config.ErrConfigNotFound)) {
				return fmt.Errorf("invalid config; %w", err)
			}

			for _, c := range c.StringSlice("set") {
				arr := strings.Split(c, "=")
				if len(arr) > 1 {
					cfg.Variables.Set(arr[0], strings.Join(arr[1:], "="))
				}
			}

			if c.Bool("debug") || cfg.Debug {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			} else {
				slog.SetLogLoggerLevel(slog.LevelInfo)
			}

			if c.Bool("quiet") {
				c.App.ErrWriter = io.Discard
				slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
				cfg.Quiet = true
			} else {
				if c.IsSet("output") {
					cfg.Output = c.String("output")
				} else if c.Bool("raw") {
					cfg.Output = output.FormatRaw
				} else if c.Bool("cockpit") {
					cfg.Output = output.FormatCockpit
				} else if cfg.Output == "" {
					cfg.Output = output.FormatPrefixed
				}
			}

			if c.Bool("dry-Run") {
				cfg.DryRun = true
			}

			if cfg.Output == output.FormatCockpit && !stdoutIsTTY() {
				cfg.Output = output.FormatPrefixed
			}

			au = aurora.NewAurora(stdoutIsTTY() && cfg.Output != output.FormatJSON)

			return nil
		},
		Action: rootAction,
		Commands: []*cli.Command{
			newRunCommand(),
			newInitCommand(),
			newListCommand(),
			newShowCommand(),
			newWatchCommand(),
			newCompletionCommand(),
			newGraphCommand(),
			newValidateCommand(),
			newSkillCommand(),
		},
		Authors: []*cli.Author{
			{
				Name:  "Yevhen Terentiev",
				Email: "yevhen.terentiev@gmail.com",
			},
		},
	}
}

func Abort() {
	cancelMu.Lock()
	defer cancelMu.Unlock()

	cancelFn()
}

func listenSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	for sig := range sigs {
		Abort()
		os.Exit(int(sig.(syscall.Signal)))
	}
}

func rootAction(c *cli.Context) (err error) {
	cancelMu.Lock()
	c.Context, cancelFn = context.WithCancel(c.Context)
	cancelMu.Unlock()

	taskRunner, err := buildTaskRunner(c.Context, c)
	if err != nil {
		return err
	}

	targets := c.Args().Slice()
	if len(targets) > 0 {
		return runTargets(targetNames(targets, false), c, taskRunner)
	}

	if nonInteractive(c) {
		return errors.New("no target specified; run 'taskctl list' to see available targets")
	}

	suggestions := buildSuggestions(cfg)
	targetSelect := promptui.Select{
		Label:        "Select task to Run",
		Items:        suggestions,
		Size:         15,
		CursorPos:    0,
		IsVimMode:    false,
		HideHelp:     false,
		HideSelected: false,
		Templates: &promptui.SelectTemplates{
			Active:   fmt.Sprintf("%s {{ .DisplayName | underline }}", promptui.IconSelect),
			Inactive: "  {{ .DisplayName }}",
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ .DisplayName | faint }}`, promptui.IconGood),
		},
		Keys: nil,
		Searcher: func(input string, index int) bool {
			return strings.Contains(suggestions[index].DisplayName, input)
		},
		StartInSearchMode: true,
	}

	fmt.Println("Please use `Ctrl-C` to exit this program.")
	index, _, err := targetSelect.Run()
	if err != nil {
		return err
	}

	selection := suggestions[index]
	if selection.IsTask {
		return runTask(cfg.Tasks[selection.Target], taskRunner)
	}

	return runPipeline(cfg.Pipelines[selection.Target], taskRunner, cfg.Summary || c.Bool("summary"))
}

func buildTaskRunner(ctx context.Context, c *cli.Context) (*runner.TaskRunner, error) {
	ta := taskArgs(c)
	variables := cfg.Variables.With("Args", strings.Join(ta, " "))
	variables.Set("ArgsList", ta)

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
			DisplayName: fmt.Sprintf("%s - %s", v, au.Gray(12, "pipeline").String()),
		})
	}

	for k, v := range cfg.Tasks {
		desc := "task"
		if v.Description != "" {
			desc = v.Description
		}
		suggestions = append(suggestions, suggestion{
			Target:      k,
			DisplayName: fmt.Sprintf("%s - %s", k, au.Gray(12, desc).String()),
			IsTask:      true,
		})
	}

	slices.SortFunc(suggestions, func(a, b suggestion) int {
		return strings.Compare(a.Target, b.Target)
	})

	return suggestions
}
