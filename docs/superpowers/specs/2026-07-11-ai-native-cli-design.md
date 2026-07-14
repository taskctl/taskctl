# AI-Native CLI Design: machine-readable interface + agent skill

**Date:** 2026-07-11
**Status:** Approved
**Goal:** Make taskctl reliably drivable by AI coding agents (Claude Code and similar) via a coherent machine-readable CLI surface, plus a `taskctl skill install` command that teaches agents how to use it.

## Motivation

AI agents drive CLIs over a shell. Today taskctl's output is human-oriented (templates, colors, spinner), `run` without arguments opens an interactive selector, and there is no structured way to discover tasks or interpret results. Agents must parse free text or read `tasks.yaml` themselves.

This design adds:
1. JSON discovery output for `list` and `show`
2. An NDJSON event stream for `run`
3. Non-interactive safety guarantees
4. A `skill install` command that installs a Claude Code Agent Skill

Out of scope (explicitly rejected during brainstorming): MCP server mode, AI task types inside pipelines, a new `describe` command (extended `show` covers it), dry-run plan output, multi-harness skill targets (Claude Code only for now).

## 1. Discovery: `list` / `show` JSON output

The existing global `--output` flag (values `raw`, `prefixed`, `cockpit`, selected in `cmd/cmd.go`) gains a fourth value: `json`. `list` and `show` — which currently ignore this flag — honor it.

### `taskctl --output json list`

Single JSON object on stdout:

```json
{
  "schema_version": 1,
  "tasks": [{"name": "build", "description": "...", "context": "local"}],
  "pipelines": [{"name": "release", "stages": ["build", "test"]}],
  "contexts": ["local", "docker"],
  "watchers": ["sources"]
}
```

- Arrays sorted by name (matches current behavior).
- The `list tasks|pipelines|watchers` subcommands emit a wrapped object with only the corresponding key (e.g. `{"schema_version": 1, "tasks": [...]}`) — a top-level object everywhere, never a bare array.

### `taskctl --output json show <task|pipeline>`

Full resolved detail for one named task or pipeline:

- Task: `name`, `description`, `context`, `commands` (array), `env`, `variables`, `dir`, `timeout`, `allow_failure`, `condition`.
- Pipeline: `name`, `stages`: array of `{name, task, depends_on: [...], condition, allow_failure}` — the DAG edges, so an agent can reason about execution order.

Unknown name → non-zero exit, error message on **stderr** (stdout stays valid JSON or empty).

### Implementation

- Shared response structs in `internal/schema/schema.go` (new package): `ListResponse`, `TaskDetail`, `PipelineDetail`, `StageDetail`. Every top-level document carries `schema_version: 1`; bump on breaking changes.
- `cmd/list.go` and `cmd/show.go` branch on the resolved output format: template path (current behavior) vs `json.Encoder` to stdout.
- `show` for pipelines is new capability (today `show` handles tasks only) — it reads `cfg.Pipelines[name]` (`*scheduler.ExecutionGraph`) and serializes nodes + `To`/`From` edges.

## 2. Run events: NDJSON stream

`taskctl --output json run <target>` emits one JSON object per line on stdout, nothing else. Human summary, spinner, and decoration are fully suppressed; taskctl's own diagnostics go to stderr.

### Event schema (v1)

| event | fields |
|---|---|
| `run_started` | `schema_version`, `targets: [string]` |
| `task_started` | `task` |
| `task_output` | `task`, `stream: "stdout"\|"stderr"`, `data` (one line, no trailing newline) |
| `task_finished` | `task`, `status: "done"\|"failed"\|"skipped"\|"canceled"`, `exit_code`, `duration_ms`, `error` (message, only on failure) |
| `run_finished` | `status: "done"\|"failed"`, `duration_ms`, `tasks: [{task, status, exit_code, duration_ms}]` |

Only `run_started` carries `schema_version` (one stream = one version). Single-task runs (`run task foo`) emit the same stream.

### Implementation

- New decorator `output/json.go`: `jsonOutputWriter` implements `DecoratedOutputWriter` — `WriteHeader` → `task_started`, `Write` → line-buffered `task_output` events (split on `\n`, flush remainder on footer), `WriteFooter` → `task_finished` (status/exit code/duration read from the `*task.Task`).
- Tasks run concurrently, so all writers share one package-level mutex-guarded `json.Encoder` on stdout; each `Encode` call is atomic → no interleaved lines.
- Stderr vs stdout attribution: `TaskOutput.Stdout()`/`Stderr()` currently funnel into one decorator; the json decorator needs two `io.Writer` facets (thin wrapper structs tagging the stream) so `task_output` events carry the correct `stream`.
- `cmd/run.go`: when format is json, emit `run_started` before scheduling and `run_finished` instead of `printSummary`; suppress the `\r\n` cosmetic write.
- Registration in `output.NewTaskOutput` switch + `FormatJSON = "json"` constant.

## 3. Non-interactive safety

- New global bool flag `--no-input`. Non-interactive mode is active when the flag is set **or** stdin is not a TTY (`mattn/go-isatty`, already an indirect dependency — promote to direct).
- When non-interactive:
  - `run` with no arguments returns an error (`no target specified; run 'taskctl list'`) instead of opening the `promptui` selector.
  - `init` errors unless its inputs are fully specified by flags.
- When **stdout** is not a TTY:
  - `cockpit` output degrades to `prefixed` (the spinner is meaningless in a pipe).
  - aurora colors are disabled (wire `aurora.NewAurora(isTTY)` through, or gate at the call sites in `cmd/`).
- `--output json` implies both: no prompts, no colors, no spinner, regardless of TTY.
- Exit codes: `0` success, non-zero failure (unchanged, but documented). Machine-readable failure detail lives in `run_finished` / stderr, not in exit-code taxonomy.

## 4. `taskctl skill install`

New `cmd/skill.go`:

```
taskctl skill install [--global] [--force]
```

- Writes an embedded (`go:embed`) `SKILL.md` template to `.claude/skills/taskctl/SKILL.md` relative to the current working directory; `--global` targets `~/.claude/skills/taskctl/SKILL.md`.
- Refuses to overwrite an existing file unless `--force`; creates directories as needed.
- Prints the installed path on success.

### SKILL.md content (generic + live discovery)

Frontmatter: `name: taskctl`, `description` triggering on taskctl config files (tasks.yaml, taskctl.yaml) and requests to run project tasks/pipelines. Body teaches:

1. Discover targets: `taskctl --output json list` (never parse tasks.yaml directly).
2. Inspect before running: `taskctl --output json show <name>`.
3. Execute: `taskctl --output json --no-input run <target>`; read the NDJSON events; `run_finished.status` is the source of truth.
4. Event schema reference (the table above, condensed).
5. Exit code semantics and stderr-for-diagnostics convention.

The skill contains no project-specific task names — it always discovers live, so it never goes stale.

## Error handling

- Invalid `--output` value: existing error path (`unknown decorator`) covers `json` typos.
- JSON encoding errors (closed pipe, etc.): fail the run with error to stderr.
- `skill install` filesystem errors (permissions, missing home dir): plain error, non-zero exit.

## Testing

- Table-driven tests in `cmd/` following the existing `*_test.go` style (run command against `testdata/*.yaml`, assert output).
- JSON assertions: unmarshal output and compare structs — not string matching.
- Concurrency test: pipeline with parallel stages under `--output json`; every stdout line must parse as JSON (catches interleaving regressions).
- Non-interactive tests: `--no-input run` with no args errors; cockpit + non-TTY writer produces prefixed output.
- `skill install` tests against a temp dir: fresh install, existing-file refusal, `--force`, `--global` path resolution.

## Documentation

README gains a "taskctl for AI agents" section: the `--output json` surface, event schema table, `--no-input`, and `taskctl skill install`.
