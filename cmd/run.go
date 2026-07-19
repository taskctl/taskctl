package cmd

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/scheduler"
	"github.com/taskctl/taskctl/task"
)

func newRunCommand(cfg *config.Config) *cobra.Command {
	runCmd := &cobra.Command{
		Use:     "run TARGET [TARGET...] [-- task-args]",
		Short:   "run one or more pipelines or tasks",
		Long:    "Runs one or more named pipelines or tasks in order, stopping at the first failure. Arguments after \"--\" are passed through to the tasks as $Args.",
		GroupID: groupRun,
		Example: "  taskctl run pipeline1\n" +
			"  taskctl run task1 task2\n" +
			"  taskctl run test -- -v",
		Args:              minArgs(1, "run requires at least one task or pipeline name"),
		ValidArgsFunction: targetCompletion(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, _ := splitArgsAtDash(cmd, args)
			// Support the legacy `run pipeline <name>` form by dropping a leading
			// "pipeline" keyword — without swallowing a real target of that name
			// elsewhere in the list.
			if len(targets) > 0 && targets[0] == "pipeline" {
				targets = targets[1:]
			}
			if len(targets) == 0 {
				return errors.New("no target specified")
			}
			return runTargets(cmd, cfg, targets, false)
		},
	}

	taskCmd := &cobra.Command{
		Use:   "task TASK [TASK...] [-- task-args]",
		Short: "run one or more tasks",
		Long:  "Runs one or more named tasks directly, rejecting pipeline names (unlike plain `run`).",
		Example: "  taskctl run task test -- -v\n" +
			"  taskctl run task task1 task2",
		Args:              minArgs(1, "run task requires at least one task name"),
		ValidArgsFunction: taskCompletion(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, _ := splitArgsAtDash(cmd, args)
			return runTargets(cmd, cfg, targets, true)
		},
	}
	runCmd.AddCommand(taskCmd)

	return runCmd
}

// taskCompletion completes task names only; `run task` rejects pipelines.
func taskCompletion(cfg *config.Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completionFunc(cfg, func() []string {
		return slices.Sorted(maps.Keys(cfg.Tasks))
	})
}

// splitArgsAtDash separates target names from the arguments after "--", which
// cobra strips but records the position of via ArgsLenAtDash.
func splitArgsAtDash(cmd *cobra.Command, args []string) (targets, passArgs []string) {
	dash := cmd.ArgsLenAtDash()
	if dash < 0 {
		return args, nil
	}
	return args[:dash], args[dash:]
}

// runTargets runs each named target in order, aggregates the executed pipeline
// graphs and directly-run tasks, and brackets the run with the run_started /
// run_finished NDJSON events (no-ops outside json mode). It stops at the first
// failing target and returns its error. When tasksOnly is set, pipeline names
// are not matched (used by `run task`).
func runTargets(cmd *cobra.Command, cfg *config.Config, targets []string, tasksOnly bool) error {
	taskRunner, err := buildTaskRunner(cmd, cfg)
	if err != nil {
		return err
	}

	summary := summaryEnabled(cmd, cfg)
	emitRunStarted(cfg, targets)

	var graphs []*scheduler.ExecutionGraph
	var tasks []*task.Task
	for _, name := range targets {
		g, t, terr := runTarget(cfg, taskRunner, name, tasksOnly)
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

	// When finishRun surfaced the failure (summary or JSON run_finished event),
	// mark it reported so the top-level presenter doesn't print it again.
	if reported := finishRun(cfg, graphs, tasks, summary, err); err != nil && reported {
		return reportedError{err}
	}
	return err
}

// runTarget runs the pipeline or task named by name and reports back
// whichever of the two it ran, so callers can aggregate results for the
// NDJSON run_finished event.
func runTarget(cfg *config.Config, taskRunner *runner.TaskRunner, name string, tasksOnly bool) (g *scheduler.ExecutionGraph, t *task.Task, err error) {
	if !tasksOnly {
		if p := cfg.Pipelines[name]; p != nil {
			if err = runPipeline(p, taskRunner); err != nil {
				return p, nil, fmt.Errorf("pipeline %q failed: %w", name, err)
			}
			return p, nil, nil
		}
	}

	t = cfg.Tasks[name]
	if t == nil {
		kind := "task or pipeline"
		if tasksOnly {
			kind = "task"
		}
		return nil, nil, fmt.Errorf("unknown %s %q", kind, name)
	}
	if err = runTask(t, taskRunner); err != nil {
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

// runTask finishes the runner even when the task fails, for the same
// teardown-before-summary reason as runPipeline.
func runTask(t *task.Task, taskRunner *runner.TaskRunner) error {
	err := taskRunner.Run(t)
	taskRunner.Finish()

	return err
}

// emitRunStarted writes the run_started NDJSON event when running in json
// output mode; it is a no-op otherwise.
func emitRunStarted(cfg *config.Config, targets []string) {
	if cfg.Output != output.FormatJSON {
		return
	}

	_ = output.EmitRunStarted(os.Stdout, targets)
}

// emitRunFinished writes the run_finished NDJSON event when running in json
// output mode; it is a no-op otherwise. It builds per-task results from both
// executed pipeline graphs and directly-run tasks, and derives an overall
// status of "failed" if err is non-nil or any task/stage failed.
func emitRunFinished(cfg *config.Config, graphs []*scheduler.ExecutionGraph, tasks []*task.Task, runErr error) {
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

// summaryEnabled reports whether the human end-of-run summary should print. An
// explicitly passed --summary wins in both directions. With no explicit flag:
// --quiet suppresses the summary, raw output defaults it off (raw is meant for
// clean, pipeable output) unless config opts in, and every other human mode
// defaults it on.
func summaryEnabled(cmd *cobra.Command, cfg *config.Config) bool {
	if cmd.Flags().Changed("summary") {
		v, _ := cmd.Flags().GetBool("summary")
		return v
	}

	if cfg.Quiet {
		return false
	}

	if cfg.Summary != nil {
		return *cfg.Summary
	}

	return cfg.Output != output.FormatRaw
}

// finishRun emits the run_finished NDJSON event (json mode) or, in a human
// output mode with summary enabled, prints the persistent end-of-run summary.
// It runs after the dashboard has torn down (via each caller's Finish), so the
// summary stays on screen. It reports whether the run's outcome was surfaced to
// the user (JSON event or a printed summary), so callers can suppress a
// duplicate top-level error line.
func finishRun(cfg *config.Config, graphs []*scheduler.ExecutionGraph, tasks []*task.Task, summary bool, err error) bool {
	emitRunFinished(cfg, graphs, tasks, err)

	if cfg.Output == output.FormatJSON {
		return true
	}
	if !summary {
		return false
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
		return false
	}

	_, _ = fmt.Fprint(os.Stdout, "\r\n")
	output.PrintRunSummary(os.Stdout, items, total)
	return true
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
			if stage.Pipeline != nil && stage.Pipeline.LastError() != nil {
				s.ErrMessage = stage.Pipeline.LastError().Error()
			}
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
