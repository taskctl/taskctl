package cmd

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/scheduler"
	"github.com/taskctl/taskctl/task"
)

func newRunCommand() *cli.Command {
	var taskRunner *runner.TaskRunner
	cmd := &cli.Command{
		Name:      "run",
		ArgsUsage: "run (PIPELINE1 OR TASK1) [PIPELINE2 OR TASK2]... [flags] [-- TASKS_ARGS]",
		Usage:     "runs pipeline or task",
		UsageText: "taskctl run pipeline1\n" +
			"taskctl pipeline1\n" +
			"taskctl task1",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "dry run",
			},
			&cli.BoolFlag{
				Name:    "summary",
				Usage:   "show summary",
				Aliases: []string{"s"},
				Value:   true,
			},
		},
		Before: func(c *cli.Context) (err error) {
			cancelMu.Lock()
			c.Context, cancelFn = context.WithCancel(c.Context)
			cancelMu.Unlock()

			taskRunner, err = buildTaskRunner(c.Context, c)
			return err
		},
		After: func(c *cli.Context) error {
			return nil
		},
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				return errors.New("no target specified")
			}

			return runTargets(targetNames(c.Args().Slice(), true), taskRunner, summaryEnabled(c))
		},
		Subcommands: []*cli.Command{
			{
				Name:      "task",
				ArgsUsage: "task (TASK1) [TASK2]... [flags] [-- TASK_ARGS]",
				Usage:     "Run specified task(s)",
				Action: func(c *cli.Context) error {
					emitRunStarted(targetNames(c.Args().Slice(), false))

					var tasks []*task.Task
					var err error
					for _, v := range c.Args().Slice() {
						if v == "--" {
							break
						}

						t := cfg.Tasks[v]
						if t == nil {
							err = fmt.Errorf("unknown task %s", v)
							break
						}
						tasks = append(tasks, t)
						if terr := runTask(t, taskRunner); terr != nil {
							err = terr
							break
						}
					}

					finishRun(nil, tasks, summaryEnabled(c), err)
					return err
				},
			},
		},
	}

	return cmd
}

// runTargets runs each named target in order, aggregates the executed pipeline
// graphs and directly-run tasks, and brackets the run with the run_started /
// run_finished NDJSON events (no-ops outside json mode). It stops at the first
// failing target and returns its error.
func runTargets(targets []string, taskRunner *runner.TaskRunner, summary bool) error {
	emitRunStarted(targets)

	var graphs []*scheduler.ExecutionGraph
	var tasks []*task.Task
	var err error

	for _, name := range targets {
		g, t, terr := runTarget(name, taskRunner)
		if g != nil {
			graphs = append(graphs, g)
		}
		if t != nil {
			tasks = append(tasks, t)
		}
		if terr != nil {
			err = terr
			break
		}
	}

	finishRun(graphs, tasks, summary, err)
	return err
}

// runTarget runs the pipeline or task named by name and reports back
// whichever of the two it ran, so callers can aggregate results for the
// NDJSON run_finished event.
func runTarget(name string, taskRunner *runner.TaskRunner) (g *scheduler.ExecutionGraph, t *task.Task, err error) {
	p := cfg.Pipelines[name]
	if p != nil {
		err = runPipeline(p, taskRunner)
		if err != nil {
			return p, nil, fmt.Errorf("pipeline %q failed: %w", name, err)
		}
		return p, nil, nil
	}

	t = cfg.Tasks[name]
	if t == nil {
		return nil, nil, fmt.Errorf("unknown task or pipeline %q", name)
	}
	err = runTask(t, taskRunner)
	if err != nil {
		return nil, t, fmt.Errorf("task %q failed: %w", name, err)
	}

	return nil, t, nil
}

// runPipeline finishes the scheduler even when the run fails: Finish tears
// down the live dashboard and runs context Down hooks, and the end-of-run
// summary must print after that teardown.
func runPipeline(g *scheduler.ExecutionGraph, taskRunner *runner.TaskRunner) error {
	sd := scheduler.NewScheduler(taskRunner)

	err := sd.Schedule(g)
	sd.Finish()

	return err
}

// targetNames extracts the list of target names from raw CLI args, stopping
// at "--" (task args follow). When skipPipelineKeyword is true, the literal
// "pipeline" token (used by `taskctl run pipeline <name>`) is dropped.
func targetNames(args []string, skipPipelineKeyword bool) []string {
	names := make([]string, 0, len(args))
	for _, v := range args {
		if v == "--" {
			break
		}
		if skipPipelineKeyword && v == "pipeline" {
			continue
		}
		names = append(names, v)
	}

	return names
}

// emitRunStarted writes the run_started NDJSON event when running in json
// output mode; it is a no-op otherwise.
func emitRunStarted(targets []string) {
	if cfg.Output != output.FormatJSON {
		return
	}

	_ = output.EmitRunStarted(os.Stdout, targets)
}

// emitRunFinished writes the run_finished NDJSON event when running in json
// output mode; it is a no-op otherwise. It builds per-task results from both
// executed pipeline graphs and directly-run tasks, and derives an overall
// status of "failed" if err is non-nil or any task/stage failed.
func emitRunFinished(graphs []*scheduler.ExecutionGraph, tasks []*task.Task, runErr error) {
	if cfg.Output != output.FormatJSON {
		return
	}

	var results []output.TaskResult
	var totalDuration time.Duration
	failed := runErr != nil

	for _, g := range graphs {
		names := slices.Sorted(maps.Keys(g.Nodes()))
		for _, name := range names {
			stage := g.Nodes()[name]
			status := stageStatus(stage)
			if status == "failed" || status == "canceled" {
				failed = true
			}

			taskName := stage.Name
			var exitCode int
			var durationMs int64
			if stage.Task != nil {
				taskName = stage.Task.Name
				exitCode = int(stage.Task.ExitCode)
				durationMs = stage.Task.Duration().Milliseconds()
			} else {
				durationMs = stage.Duration().Milliseconds()
			}

			results = append(results, output.TaskResult{
				Task:       taskName,
				Status:     status,
				ExitCode:   exitCode,
				DurationMs: durationMs,
			})
		}
		totalDuration += g.Duration()
	}

	for _, t := range tasks {
		status := output.TaskStatus(t)
		if status == "failed" {
			failed = true
		}

		results = append(results, output.TaskResult{
			Task:       t.Name,
			Status:     status,
			ExitCode:   int(t.ExitCode),
			DurationMs: t.Duration().Milliseconds(),
		})
		totalDuration += t.Duration()
	}

	status := "done"
	if failed {
		status = "failed"
	}

	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
	}

	_ = output.EmitRunFinished(os.Stdout, status, totalDuration.Milliseconds(), results, errMsg)
}

// stageStatus maps a scheduler stage status to the NDJSON status vocabulary.
func stageStatus(stage *scheduler.Stage) string {
	switch stage.ReadStatus() {
	case scheduler.StatusDone:
		return "done"
	case scheduler.StatusError:
		return "failed"
	case scheduler.StatusSkipped:
		return "skipped"
	default:
		return "canceled"
	}
}

// runTask finishes the runner even when the task fails, for the same
// teardown-before-summary reason as runPipeline.
func runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t)
	taskRunner.Finish()

	return err
}

func taskArgs(c *cli.Context) []string {
	var runArgs []string
	var dash = -1
	for k, arg := range c.Args().Slice() {
		if arg == "--" {
			dash = k
		}
	}

	if dash >= 0 && dash != c.NArg()-1 {
		runArgs = c.Args().Slice()[dash+1:]
	}

	return runArgs
}

// summaryEnabled reports whether the human end-of-run summary should print.
// An explicitly passed --summary wins in both directions; the lineage walk is
// required because the flag is declared on both the app root and the run
// command, and c.Bool alone would resolve the nearest (possibly unset,
// default-true) declaration, silently ignoring an explicit root-level value.
// With no explicit flag: --quiet suppresses the summary, raw output defaults
// it off (raw is meant for clean, pipeable output) unless config opts in, and
// every other human mode defaults it on.
func summaryEnabled(c *cli.Context) bool {
	for _, ctx := range c.Lineage() {
		if ctx.IsSet("summary") {
			return ctx.Bool("summary")
		}
	}

	if cfg.Quiet {
		return false
	}

	return cfg.Summary || cfg.Output != output.FormatRaw
}

// finishRun emits the run_finished NDJSON event (json mode) or, in a human
// output mode with summary enabled, prints the persistent end-of-run summary.
// It runs after the dashboard has torn down (via each caller's Finish), so the
// summary stays on screen.
func finishRun(graphs []*scheduler.ExecutionGraph, tasks []*task.Task, summary bool, err error) {
	emitRunFinished(graphs, tasks, err)

	if cfg.Output == output.FormatJSON || !summary {
		return
	}

	var items []output.StageSummary
	var total time.Duration
	for _, g := range graphs {
		items = append(items, summarizeGraph(g)...)
		total += g.Duration()
	}
	for _, t := range tasks {
		total += t.Duration()
	}
	items = append(items, output.SummarizeTasks(tasks)...)

	if len(items) == 0 {
		return
	}

	_, _ = fmt.Fprint(os.Stdout, "\r\n")
	output.PrintRunSummary(os.Stdout, items, total)
}

func summarizeGraph(g *scheduler.ExecutionGraph) []output.StageSummary {
	names := slices.Sorted(maps.Keys(g.Nodes()))
	items := make([]output.StageSummary, 0, len(names))
	for _, name := range names {
		stage := g.Nodes()[name]

		var s output.StageSummary
		if stage.Task != nil {
			s = output.SummarizeTask(stage.Task)
		} else {
			s.Start = stage.Start
			s.Duration = stage.Duration()
		}
		s.Name = stage.Name
		s.Status = stageStatus(stage)
		// A condition-skipped task still leaves its stage Done — report the
		// task-level "skipped" instead of success.
		if s.Status == "done" && stage.Task != nil && stage.Task.Skipped {
			s.Status = "skipped"
		}

		items = append(items, s)
	}
	return items
}
