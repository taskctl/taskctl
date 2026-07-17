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

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/tui"
)

var stdin io.ReadCloser
var cancelFn func()
var cancelMu sync.Mutex
var cfg *config.Config

func Stdin() io.ReadCloser {
	return stdin
}

func SetStdin(newStdin io.ReadCloser) {
	stdin = newStdin
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
				Usage:   "output format (raw, prefixed or cockpit)",
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
		for _, target := range targets {
			if target == "--" {
				break
			}

			err = runTarget(target, c, taskRunner)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if !tui.Interactive(stdin) {
		return errors.New("no task or pipeline specified")
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
			DisplayName: fmt.Sprintf("%s - %s", v, "pipeline"),
		})
	}

	for k, v := range cfg.Tasks {
		desc := "task"
		if v.Description != "" {
			desc = v.Description
		}
		suggestions = append(suggestions, suggestion{
			Target:      k,
			DisplayName: fmt.Sprintf("%s - %s", k, desc),
			IsTask:      true,
		})
	}

	slices.SortFunc(suggestions, func(a, b suggestion) int {
		return strings.Compare(a.Target, b.Target)
	})

	return suggestions
}
