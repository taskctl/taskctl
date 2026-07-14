# AI-Native CLI Implementation Plan

> **For agentic workers:** Execute stage-by-stage with plan-executor subagents. Each stage is independently testable. Steps use checkbox (`- [ ]`) syntax for tracking. Spec: `docs/superpowers/specs/2026-07-11-ai-native-cli-design.md`.

**Goal:** Machine-readable CLI surface for AI agents (JSON discovery, NDJSON run events, non-interactive safety) plus `taskctl skill install` for Claude Code.

**Architecture:** The existing global `--output` flag gains a `json` value. `list`/`show` branch to JSON encoders using shared structs in a new `internal/schema` package. A new `output/json.go` decorator implements the existing `DecoratedOutputWriter` interface and streams NDJSON events through a mutex-guarded shared encoder. Non-interactive behavior is gated on a new `--no-input` flag plus TTY detection. `skill install` writes an embedded SKILL.md.

**Tech Stack:** Go 1.23, `urfave/cli/v2`, `encoding/json`, `mattn/go-isatty` (promote from indirect dep), `go:embed`.

## Global Constraints

- **Never commit.** Leave all changes in the working tree; the user commits manually.
- Every stage must end with `go build ./...`, `go test ./...`, and `golangci-lint run` passing. Report any failures back to the main model verbatim — do NOT attempt large refactors to fix them.
- Follow existing code style: table-driven tests as in `cmd/cmd_test.go` (see its `makeTestApp`/run helpers before writing tests), stdlib `errors`/`fmt` error wrapping, no new third-party deps beyond promoting `mattn/go-isatty`.
- JSON documents/events on **stdout** only; diagnostics on stderr. Every top-level discovery document carries `"schema_version": 1`. NDJSON streams carry `schema_version` only on `run_started`.
- JSON field names are `snake_case` via struct tags.
- Do not rename existing flags/commands (including the oddly-capitalized `Run`/`dry-Run` names).

---

### Stage 1: `internal/schema` package + `FormatJSON` constant

**Files:**
- Create: `internal/schema/schema.go`
- Create: `internal/schema/schema_test.go`
- Modify: `output/output.go` (add constant), `cmd/cmd.go:86` (flag usage string)

**Interfaces (Produces):**
- `output.FormatJSON = "json"` constant alongside `FormatRaw`/`FormatPrefixed`/`FormatCockpit` in `output/output.go`. Do NOT add it to the `NewTaskOutput` switch yet (Stage 3 does that).
- Package `internal/schema` with exported types (all fields with `json:"snake_case"` tags):
  - `ListResponse{SchemaVersion int, Tasks []TaskSummary, Pipelines []PipelineSummary, Contexts []string, Watchers []string}` — `Tasks`/`Pipelines`/`Contexts`/`Watchers` marshal as `[]` not `null` when empty (initialize slices).
  - `TaskSummary{Name, Description, Context string}`
  - `PipelineSummary{Name string, Stages []string}`
  - `TaskDetail{Name, Description, Context string, Commands []string, Env map[string]string, Variables map[string]string, Dir string, TimeoutSeconds *float64, AllowFailure bool, Condition string}` — omitempty on optional fields.
  - `PipelineDetail{Name string, Stages []StageDetail}`
  - `StageDetail{Name, Task string, DependsOn []string, Condition string, AllowFailure bool}`
- Constructor funcs:
  - `NewTaskSummary(t *task.Task) TaskSummary`
  - `NewTaskDetail(t *task.Task) TaskDetail` — env/variables via `t.Env.Map()` / `t.Variables.Map()` (returns `map[string]any`; stringify values with `fmt.Sprint` into `map[string]string`), timeout from `t.Timeout.Seconds()` when non-nil.
  - `NewPipelineDetail(name string, g *scheduler.ExecutionGraph) PipelineDetail` — stages from `g.Nodes()` sorted by name; `DependsOn` from `Stage.DependsOn`; `Task` is `stage.Task.Name` when `stage.Task != nil`, else empty.
  - `NewPipelineSummary(name string, g *scheduler.ExecutionGraph) PipelineSummary` — stage names sorted.

**Update flag usage** in `cmd/cmd.go`: `"output format (raw, prefixed, cockpit or json)"`.

- [ ] Write failing tests in `schema_test.go`: build a `*task.Task` and a small `*scheduler.ExecutionGraph` (two stages, one `depends_on`), assert constructor output including JSON round-trip field names (`json.Marshal` → check `snake_case` keys, `[]` for empty slices).
- [ ] Run `go test ./internal/schema/` — expect FAIL (package missing).
- [ ] Implement `schema.go` and the `output.FormatJSON` constant; update the flag usage string.
- [ ] Run `go build ./... && go test ./... && golangci-lint run` — expect PASS.

---

### Stage 2: JSON output for `list` and `show`

**Files:**
- Modify: `cmd/list.go`, `cmd/show.go`
- Test: `cmd/list_test.go`, `cmd/show_test.go`

**Interfaces:**
- Consumes: `internal/schema` types/constructors, `output.FormatJSON`, package-level `cfg *config.Config` (its `Output` field holds the resolved format after the app `Before` hook).
- Produces: no new exported API; behavior only.

**Behavior:**
- `taskctl --output json list`: encode a `schema.ListResponse` to stdout with `json.Encoder`. Tasks sorted by name with `NewTaskSummary`; pipelines via `NewPipelineSummary`; contexts/watchers are sorted name arrays (reuse existing `utils.MapKeys` + `sort.Strings`). Text template path unchanged when format ≠ json.
- `list tasks|pipelines|watchers` subcommands with json: emit a wrapped object containing `schema_version` and only the corresponding key (e.g. `{"schema_version":1,"tasks":[...]}`). Reuse `ListResponse` with only that slice populated and add `omitempty` on the other slices — OR define per-subcommand inline structs; pick whichever keeps `list` top-level output unchanged (top-level `list` must always include all four keys, so if using `omitempty`, initialize all four slices there). State the choice in the stage report.
- `taskctl --output json show <name>`: look up `cfg.Tasks[name]` first, then `cfg.Pipelines[name]`; encode `TaskDetail` or `PipelineDetail`, wrapped with a version: emit `{"schema_version":1,"task":{...}}` or `{"schema_version":1,"pipeline":{...}}`. Unknown name → `return fmt.Errorf("unknown task or pipeline %s", name)` (error goes to stderr, non-zero exit). Text mode behavior unchanged (tasks only, template).

- [ ] Write failing tests: run the test app with `-o json list` / `show` against existing fixtures in `cmd/testdata/` (follow the capture pattern in existing `cmd/*_test.go`), `json.Unmarshal` the output and assert fields — never string-compare JSON.
- [ ] Run `go test ./cmd/` — expect FAIL.
- [ ] Implement the branches in `list.go` and `show.go`.
- [ ] Run `go build ./... && go test ./... && golangci-lint run` — expect PASS.

---

### Stage 3: NDJSON run event stream (`output/json.go` + run wiring)

**Files:**
- Create: `output/json.go`, `output/json_test.go`
- Modify: `output/output.go` (switch + stream-aware writers), `cmd/run.go`, `cmd/cmd.go` (`rootAction`)
- Test: `cmd/run_test.go`

**Interfaces (Produces), all in package `output`:**
- Event structs (snake_case tags): `RunStartedEvent{Event string, SchemaVersion int, Targets []string}`, `TaskStartedEvent{Event, Task string}`, `TaskOutputEvent{Event, Task, Stream, Data string}`, `TaskFinishedEvent{Event, Task, Status string, ExitCode int, DurationMs int64, Error string ",omitempty"}`, `RunFinishedEvent{Event, Status string, DurationMs int64, Tasks []TaskResult}`, `TaskResult{Task, Status string, ExitCode int, DurationMs int64}`.
- `func EmitRunStarted(w io.Writer, targets []string) error` and `func EmitRunFinished(w io.Writer, status string, durationMs int64, results []TaskResult) error` — used by `cmd`.
- All event writes (decorator + Emit funcs) go through one package-level helper `writeEvent(w io.Writer, v any) error` guarded by a package-level `sync.Mutex`: marshal first, then a single `w.Write` of `append(data, '\n')` — atomicity is the point; concurrent tasks share stdout.
- `newJSONOutputWriter(t *task.Task, w io.Writer) *jsonOutputWriter` implementing `DecoratedOutputWriter`:
  - `WriteHeader()` → `task_started` event.
  - `Write(p)` → buffer, split on `\n`, one `task_output` event per complete line (default stream `"stdout"`).
  - `WriteFooter()` → flush buffered remainder, then `task_finished` with status derived from the task: `t.Skipped` → `"skipped"`, `t.Errored` → `"failed"`, else `"done"`; `exit_code` from `t.ExitCode`, `duration_ms` from `t.Duration().Milliseconds()`, `error` from `t.ErrorMessage()` when failed.
- Stream attribution: add optional interface `type streamAwareWriter interface{ StreamWriter(stream string) io.Writer }` in `output/output.go`. `TaskOutput.Stdout()`/`Stderr()` type-assert the decorator; when it implements it, multi-write to `decorator.StreamWriter("stdout"|"stderr")` instead of the decorator itself. `jsonOutputWriter.StreamWriter` returns a facet that line-buffers per stream (two independent buffers so stdout/stderr writes don't corrupt each other's partial lines).
- Register `FormatJSON` in the `NewTaskOutput` switch.

**cmd wiring (Consumes the above):**
- In `cmd/run.go` add unexported helpers used by both the `Run` command action and `rootAction`:
  - `emitRunStarted(targets []string)` — no-op unless `cfg.Output == output.FormatJSON`.
  - `emitRunFinished(graphs []*scheduler.ExecutionGraph, tasks []*task.Task, err error)` — builds `[]output.TaskResult` from graph nodes (stage status mapping: `StatusDone`→done, `StatusError`→failed, `StatusSkipped`→skipped, `StatusCanceled`→canceled, anything else→canceled) and/or directly-run tasks; overall `status` is `"failed"` if err != nil or any task failed, else `"done"`; duration summed from graph `Duration()` / task durations.
  - Call sites: the `run` command `Action` and `rootAction` wrap their target loop with these; in json mode `printSummary` and the cosmetic `fmt.Fprint(os.Stdout, "\r\n")` in `runPipeline` are skipped (pass a flag or check `cfg.Output` inside `runPipeline`).

- [ ] Write failing tests in `output/json_test.go`: decorator emits valid NDJSON (unmarshal each line); multi-line and partial-line `Write` sequences; stderr facet tags `"stream":"stderr"`; footer flushes remainder.
- [ ] Write failing test in `cmd/run_test.go`: run a pipeline fixture with parallel stages (add one to `cmd/testdata/` if none exists) under `-o json`; assert **every** stdout line parses as JSON, first event is `run_started` with `schema_version`, last is `run_finished` with per-task results.
- [ ] Run `go test ./output/ ./cmd/` — expect FAIL.
- [ ] Implement `output/json.go`, the `streamAwareWriter` hook, and the cmd wiring.
- [ ] Run `go build ./... && go test ./... && golangci-lint run` — expect PASS.

---

### Stage 4: Non-interactive safety

**Files:**
- Modify: `cmd/cmd.go`, `cmd/init.go`, `go.mod` (promote `mattn/go-isatty` to direct)
- Test: `cmd/cmd_test.go`, `cmd/init_test.go`

**Interfaces (Produces):**
- Global flag `&cli.BoolFlag{Name: "no-input", Usage: "disable interactive prompts", EnvVars: []string{"TASKCTL_NO_INPUT"}}` in `NewApp`.
- Unexported helpers in `cmd/cmd.go`:
  - `func nonInteractive(c *cli.Context) bool` — true when `c.Bool("no-input")`, or `cfg.Output == output.FormatJSON`, or stdin is not a TTY (`!isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(...)`). For testability, route the isatty check through a package-level `var stdinIsTTY = func() bool {...}` that tests can stub.
  - Package-level `var au aurora.Aurora`, initialized in the app `Before` hook: `aurora.NewAurora(colorsEnabled)` where colors are enabled only if stdout is a TTY and format ≠ json. Replace package-func aurora call sites in `cmd/` (`buildSuggestions` in `cmd.go`, `printSummary` in `run.go`) with `au`.
- Behavior changes, all in the `Before` hook or action heads:
  - `rootAction` with no targets and `nonInteractive(c)` → `return errors.New("no target specified; run 'taskctl list' to see available targets")` instead of the promptui selector.
  - `init` with `nonInteractive(c)`: skip the file-name `promptui.Select` (default to `config.DefaultFileNames[0]`) and **error** instead of the overwrite confirmation when the file exists (message tells the user to remove the file or run interactively).
  - Cockpit degrade: in `Before`, after format resolution, if `cfg.Output == output.FormatCockpit` and stdout is not a TTY → `cfg.Output = output.FormatPrefixed`.

- [ ] Write failing tests: `--no-input` root invocation with no args returns the error (no prompt hang); `init --no-input` in an empty temp dir creates `config.DefaultFileNames[0]`; `init --no-input` with existing file errors; cockpit degrades under stubbed non-TTY.
- [ ] Run `go test ./cmd/` — expect FAIL.
- [ ] Implement; run `go mod tidy` to promote isatty.
- [ ] Run `go build ./... && go test ./... && golangci-lint run` — expect PASS.

---

### Stage 5: `taskctl skill install`

**Files:**
- Create: `cmd/skill.go`, `cmd/skill/SKILL.md` (embedded asset), `cmd/skill_test.go`
- Modify: `cmd/cmd.go` (register `newSkillCommand()` in `Commands`)

**Interfaces (Produces):**
- `func newSkillCommand() *cli.Command` — `skill` command with `install` subcommand, flags `--global` (bool) and `--force` (bool).
- `func installSkill(baseDir string, force bool) (string, error)` — writes the embedded SKILL.md to `<baseDir>/.claude/skills/taskctl/SKILL.md` (`os.MkdirAll` with 0o755, file 0o644); returns the written path; errors with `"skill already installed at %s (use --force to overwrite)"` when the file exists and !force. `install` action resolves baseDir: `--global` → `os.UserHomeDir()`, else `os.Getwd()`; prints `installed: <path>` on success.
- Asset embedded via `//go:embed skill/SKILL.md` into `var skillTemplate string`.

**SKILL.md content (verbatim, this is the deliverable asset):**

```markdown
---
name: taskctl
description: Run project tasks and pipelines with taskctl. Use when the project contains tasks.yaml or taskctl.yaml, or when asked to build/test/deploy via project task definitions.
---

# Using taskctl

taskctl is a concurrent task runner. Tasks and pipelines are defined in the
project config (tasks.yaml / taskctl.yaml), but do NOT parse those files —
use the machine-readable CLI instead.

## Discover what exists

    taskctl --output json list

Returns `{schema_version, tasks, pipelines, contexts, watchers}`. Task entries
carry `name`, `description`, `context`; pipeline entries carry `name` and
`stages`.

## Inspect before running

    taskctl --output json show <task-or-pipeline>

Tasks: resolved `commands`, `env`, `variables`, `dir`, `timeout_seconds`,
`allow_failure`, `condition`. Pipelines: `stages` with `depends_on` edges
(the execution DAG).

## Execute

    taskctl --output json --no-input run <target>

Stdout is an NDJSON event stream — one JSON object per line:

| event | key fields |
|---|---|
| run_started | schema_version, targets |
| task_started | task |
| task_output | task, stream (stdout/stderr), data (one line) |
| task_finished | task, status (done/failed/skipped/canceled), exit_code, duration_ms, error |
| run_finished | status (done/failed), duration_ms, tasks[] |

`run_finished.status` is the source of truth for success. Exit code is 0 on
success, non-zero on failure. taskctl's own diagnostics go to stderr.

## Rules

- Always pass `--output json --no-input`.
- Never invoke interactive commands (`taskctl` with no arguments opens a selector when in a terminal).
- Prefer running a pipeline over hand-sequencing its tasks — taskctl handles ordering and concurrency.
```

- [ ] Write failing tests in `cmd/skill_test.go` against `t.TempDir()`: fresh install creates the file with the embedded content; second install errors mentioning `--force`; `--force` overwrites; returned path is correct.
- [ ] Run `go test ./cmd/` — expect FAIL.
- [ ] Implement `cmd/skill.go` + asset; register the command.
- [ ] Run `go build ./... && go test ./... && golangci-lint run` — expect PASS.

---

### Stage 6: README documentation

**Files:**
- Modify: `README.md` (add a `## taskctl for AI agents` section after the Features section)

Content: `--output json` for `list`/`show` (one example each with abbreviated output), the NDJSON event table from the spec, `--no-input` semantics (flag, auto TTY detection, cockpit degradation), and `taskctl skill install [--global] [--force]`. Keep examples consistent with the actual schemas implemented in Stages 1–3 (read `internal/schema/schema.go` and `output/json.go` to confirm field names).

- [ ] Write the section; verify every documented field name exists in the code.
- [ ] Run `go build ./... && go test ./...` — expect PASS (docs-only, sanity check).

---

### Stage 7: Code review (main model — NOT a subagent)

Main model runs the `/code-review` skill over the full working-tree diff and fixes confirmed findings itself.

### Stage 8: Simplify pass (main model — NOT a subagent)

Main model runs the `/simplify` skill over the changed code after review fixes.
