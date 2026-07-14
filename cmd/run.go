package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/output"
	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/scheduler"
	"github.com/taskctl/taskctl/task"
	"github.com/taskctl/taskctl/utils"
)

func newRunCommand() *cli.Command {
	var taskRunner *runner.TaskRunner
	cmd := &cli.Command{
		Name:      "Run",
		ArgsUsage: "Run (PIPELINE1 OR TASK1) [PIPELINE2 OR TASK2]... [flags] [-- TASKS_ARGS]",
		Usage:     "runs pipeline or task",
		UsageText: "taskctl Run pipeline1\n" +
			"taskctl pipeline1\n" +
			"taskctl task1",
		Flags: []cli.Flag{
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
				return fmt.Errorf("no target specified")
			}

			return runTargets(targetNames(c.Args().Slice(), true), c, taskRunner)
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

					emitRunFinished(nil, tasks, err)
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
func runTargets(targets []string, c *cli.Context, taskRunner *runner.TaskRunner) error {
	emitRunStarted(targets)

	var graphs []*scheduler.ExecutionGraph
	var tasks []*task.Task
	var err error

	for _, name := range targets {
		g, t, terr := runTarget(name, c, taskRunner)
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

	emitRunFinished(graphs, tasks, err)
	return err
}

// runTarget runs the pipeline or task named by name and reports back
// whichever of the two it ran, so callers can aggregate results for the
// NDJSON run_finished event.
func runTarget(name string, c *cli.Context, taskRunner *runner.TaskRunner) (g *scheduler.ExecutionGraph, t *task.Task, err error) {
	p := cfg.Pipelines[name]
	if p != nil {
		err = runPipeline(p, taskRunner, cfg.Summary || c.Bool("summary"))
		if err != nil {
			return p, nil, fmt.Errorf("pipeline %s failed: %w", name, err)
		}
		return p, nil, nil
	}

	t = cfg.Tasks[name]
	if t == nil {
		return nil, nil, fmt.Errorf("unknown task or pipeline %s", name)
	}
	err = runTask(t, taskRunner)
	if err != nil {
		return nil, t, fmt.Errorf("task %s failed: %w", name, err)
	}

	return nil, t, nil
}

func runPipeline(g *scheduler.ExecutionGraph, taskRunner *runner.TaskRunner, summary bool) error {
	sd := scheduler.NewScheduler(taskRunner)
	//go func() {
	//	<-cancelCh
	//	sd.Cancel()
	//}()

	err := sd.Schedule(g)
	if err != nil {
		return err
	}
	sd.Finish()

	if cfg.Output != output.FormatJSON {
		_, _ = fmt.Fprint(os.Stdout, "\r\n")

		if summary {
			printSummary(g)
		}
	}

	return nil
}

// targetNames extracts the list of target names from raw CLI args, stopping
// at "--" (task args follow). When skipPipelineKeyword is true, the literal
// "pipeline" token (used by `taskctl Run pipeline <name>`) is dropped.
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
		names := utils.MapKeys(g.Nodes())
		sort.Strings(names)
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

	_ = output.EmitRunFinished(os.Stdout, status, totalDuration.Milliseconds(), results)
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

func runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t)
	if err != nil {
		return err
	}

	taskRunner.Finish()

	return nil
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

func printSummary(g *scheduler.ExecutionGraph) {
	var stages = make([]*scheduler.Stage, 0)
	for _, stage := range g.Nodes() {
		stages = append(stages, stage)
	}

	sort.Slice(stages, func(i, j int) bool {
		return stages[j].Start.Nanosecond() > stages[i].Start.Nanosecond()
	})

	_, _ = fmt.Fprintln(os.Stdout, au.Bold("Summary:").String())

	var log string
	for _, stage := range stages {
		switch stage.ReadStatus() {
		case scheduler.StatusDone:
			_, _ = fmt.Fprintln(os.Stdout, au.Sprintf(au.Green("- Stage %s was completed in %s"), stage.Name, stage.Duration()))
		case scheduler.StatusSkipped:
			_, _ = fmt.Fprintln(os.Stdout, au.Sprintf(au.Green("- Stage %s was skipped"), stage.Name))
		case scheduler.StatusError:
			log = strings.TrimSpace(stage.Task.ErrorMessage())
			_, _ = fmt.Fprintln(os.Stdout, au.Sprintf(au.Red("- Stage %s failed in %s"), stage.Name, stage.Duration()))
			if log != "" {
				_, _ = fmt.Fprintln(os.Stdout, au.Sprintf(au.Red("  > %s"), log))
			}
		case scheduler.StatusCanceled:
			_, _ = fmt.Fprintln(os.Stdout, au.Sprintf(au.Gray(12, "- Stage %s was cancelled"), stage.Name))
		default:
			_, _ = fmt.Fprintln(os.Stdout, au.Sprintf(au.Red("- Unexpected status %d for stage %s"), stage.Status, stage.Name))
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, au.Sprintf("%s: %s", au.Bold("Total duration"), au.Green(g.Duration())))
}
