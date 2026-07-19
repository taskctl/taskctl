# Stream B — Rich persistent end-of-run summary

## Context

The `cockpit` output format is being renamed to **`default`** and made the default on a TTY (Stream A). Because that dashboard shows only a spinner + "Finished" lines — never command stdout — the end-of-run **summary is the primary artifact** users are left with once the program exits. Today it is thin (`printSummary`, `cmd/run.go:316-346`), pipeline-only, and printed after the dashboard tears down. This stream substantially enriches it and extends it to single-task runs.

Facts established during exploration (verified against source):

- Summary today: `printSummary` `cmd/run.go:316-346`, called from `runPipeline` `run.go:152-174` **after** the dashboard exits (guarded `cfg.Output != FormatJSON`); pipelines only — `runTask` (`run.go:289-298`) never prints one.
- The teardown ordering already lands *after* the dashboard: `sd.Finish()` → `output.Close()` blocks on bubbletea shutdown, so anything printed after persists on screen (this is item #2's mechanism — "output disappears on exit").
- `emitRunFinished` (`run.go:208-273`) already walks `g.Nodes()` for the NDJSON path — the same traversal the summary needs (status via `stage.ReadStatus()`, duration `stage.Duration()`, task fields from `stage.Task`: `ExitCode`, `Log` sizes, `ErrorMessage()`).
- `output.TaskStatus` (`internal/output/json.go:110-119`) maps a single `*task.Task` (Skipped/Errored) to the NDJSON status vocabulary — reuse its logic for `summarizeTasks`.

User decision: summary = counts header + per-task status/duration + times + exit codes + output size + failed-task log tails + everything else we know, extended to single tasks. `--output json` never prints a human summary; `--summary=false` suppresses it.

> Note on item #4 (`--summary` "ignored in cockpit mode"): exploration showed the summary *is* printed in cockpit mode — the real defect was flag placement (`<task> --summary=false` swallowed as a target by urfave). **Stream A** fixes the parsing; **this stream** makes the summary worth keeping.

---

## Stream B — Rich persistent end-of-run summary

*(Independent of A at the design level; if done after A, hooks are in cobra files — same locations, now under `cmd/run/`.)*

### B1. Summary model — new `cmd/run/summary.go`

- `type stageSummary struct { Name string; Status int32; Start time.Time; Duration time.Duration; ExitCode int16; OutputBytes int; ErrMessage string; LogTail []string }`
- Builders (both sort by `Start`):
  - `summarizeGraph(g *scheduler.ExecutionGraph) []stageSummary` — walk `g.Nodes()` (per `emitRunFinished`'s pattern at `run.go:208-273`): status via `stage.ReadStatus()`, duration `stage.Duration()`, task fields from `stage.Task` when non-nil (`ExitCode`, `Log` sizes, `ErrorMessage()`); sub-pipeline stages report name + status + duration only.
  - `summarizeTasks(tasks []*task.Task) []stageSummary` — for direct `run task` / single-target runs; map `Skipped/Errored` → statuses like `output.TaskStatus` (`internal/output/json.go:110-119`).
- `func lastLines(buf *bytes.Buffer, n int) []string` — tail helper (trim trailing blank lines).

### B2. Renderer

- `func printRunSummary(items []stageSummary, total time.Duration)` replaces `printSummary` (`run.go:316-346`), styled with `tui.StyleBold/StyleSuccess/StyleError/StyleFaint`:
  - **Counts header**: `✔ 3 succeeded · ✗ 1 failed · ⊘ 1 skipped · 4.2s total`.
  - **Per stage**: `✔ build       1.2s` / `✗ test        3.4s  exit 2  (12 KB output)` / `⊘ deploy      skipped` — aligned name column, duration, exit code on failure, humanized captured-output size (`len(t.Log.Stdout)+len(t.Log.Stderr)`).
  - **Failed stages**: error message + last **10** lines of `Log.Stderr` (fallback `Log.Stdout`), indented, faint.
  - Footer total duration (`g.Duration()` / summed for task runs).

### B3. Hooks

- `runPipeline` (`run.go:152-174`): call `printRunSummary(summarizeGraph(g), g.Duration())` under the existing `summary && cfg.Output != FormatJSON` guard (this ordering already lands *after* dashboard teardown — see Context — so the summary persists on screen; item #2's mechanism).
- **Single tasks** (user-selected): thread `summary bool` into `runTask` (`run.go:289-298`) and print `printRunSummary(summarizeTasks(...), total)` after the run — same JSON guard. Update both call sites (root dispatch, interactive selector at `cmd.go:258-262`, `run task` subcommand).

### B4. Tests

- Table tests in `cmd/run/summary_test.go`: `summarizeGraph` status/exit/tail mapping, `lastLines` edge cases (empty buffer, fewer than N lines, trailing newline), renderer golden-ish assertions on a plain-color writer.

**Verification B**: TTY run of a pipeline with a deliberately failing task → dashboard exits, summary shows counts, per-stage lines, exit code, stderr tail; `go run . <single-task>` shows a summary too; `--summary=false` suppresses it; `--output json` never prints it.

---

## Cross-cutting (applies to all streams)

- Commit this doc with the stream's changes.
- Run tests/linters once at the end of the stream — `go run . --output json --no-input prepare` + `go test -race ./...`; tree must be clean afterwards.
- End with a main-model code review + simplify pass.
- Docs-sync agent before the PR. No branch prefixes. No commits unless explicitly requested.
