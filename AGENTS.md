# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What this is

`taskctl` is a concurrent task runner / Make alternative. Tasks and pipelines are declared in a
human-readable config (`tasks.yaml`/`taskctl.yaml`, also JSON/TOML). It is a CLI application; the
`runner`, `scheduler`, `task`, `executor`, and `variables` packages hold the reusable core, while
CLI-only support (interactive prompts and output rendering) lives under `internal/`.

## Commands

There is no Makefile. The repo **dogfoods itself**: routine work (test, lint, format, completers)
runs through taskctl's own task definitions in `tasks.yaml`, driven from the current source — so
every dev loop also exercises the tool being changed. Use the machine-readable interface described
by the bundled taskctl skill (`.agents/skills/taskctl/SKILL.md`); don't parse `tasks.yaml` or
hand-sequence its commands in bash.

```bash
go run . --output json list                       # discover tasks/pipelines (always current source)
go run . --output json show <task-or-pipeline>    # inspect resolved commands and the stage DAG
go run . --output json --no-input <target>        # run; run_finished.status is the source of truth
```

`go run .` is the default invocation — it can never run a stale binary. For repeated runs, build
once and use the binary (`go run . --output json --no-input build-host`, then `./bin/taskctl ...`);
rebuild after changing source.

Common targets: `prepare` (tidy → test → format → lint → completers; run before wrapping up),
`test`, `golangci-lint`, `fixcs`, `build-host` (host binary for dogfooding), `build`
(cross-platform release binaries), `update-completers`. `list` gives the authoritative set.

Raw Go commands — fallback for cases with no matching task, or to mirror CI exactly:

```bash
go test -race ./...                # race detector (CI runs -v -race; no task for this)
go test -run TestName ./runner/    # single test in one package
golangci-lint run                  # lint directly (golangci-lint v2; config in .golangci.yml)
go build -o bin/taskctl .          # host build without taskctl (same as build-host task)
```

CI (`.github/workflows/pull-request-checks.yml`) gates PRs on `golangci-lint` + `go test -v -race ./...`.
Go version is pinned in `go.mod` (currently 1.26). Release is handled by GoReleaser on tag push.

## Architecture

Execution flows through two layers — a pipeline DAG on top, single-task compilation underneath.

**Entry** — `main.go` → `cmd.Run(version)` builds a `urfave/cli/v2` app (`cmd/cmd.go`). Subcommands
live in `cmd/*.go` (run, init, list, show, watch, completion, graph, validate, skill). The root action
with no target opens an interactive `huh` selector (via `internal/tui`). A background goroutine
(`listenSignals`) turns SIGINT/SIGTERM into a context cancel.

**Config** — `internal/config`. `Loader` (`loader.go`) reads YAML/JSON/TOML, resolves `import:`
entries (local files, directories, or remote URLs), and merges them with `dario.cat/mergo`. Raw maps
are decoded into typed structs via `go-viper/mapstructure/v2`. The result is a `config.Config`
holding `Tasks`, `Pipelines` (already built into `scheduler.ExecutionGraph`s), `Contexts`, `Watchers`,
and a `Variables` container.

**Pipelines / scheduling** — `scheduler`. A pipeline is an `ExecutionGraph`: a DAG whose nodes are
`Stage`s and edges are `depends_on` relationships (cycle detection in `graph.go`). `Scheduler.Schedule`
polls the graph on a 50ms tick; any `StatusWaiting` stage whose deps are all `Done`/`Skipped` is
launched in its own goroutine, giving concurrent execution while respecting dependencies. Stages can
nest sub-pipelines. `AllowFailure` and per-stage `Condition` gate propagation.

**Single task** — `runner.TaskRunner.Run` (`runner/runner.go`) is the core. For each task it: resolves
the `ExecutionContext` (running `Up`/`Before` hooks), merges env + variables, checks the task
`condition`, runs `before` commands, then calls `TaskCompiler.CompileTask`.

**Compilation** — `runner/compiler.go`. `CompileTask` renders variable templates and expands
`variations` into a **linked list of `executor.Job`s** (`job.Next`). Each command becomes one job; a
task with N commands × M variations produces N×M chained jobs. Output of one command is fed to the
next as the `Output` variable.

**Execution** — `executor`. `DefaultExecutor.Execute` renders the command template (`internal/tmpl`,
Go `text/template`), parses it with `mvdan.cc/sh/v3/syntax`, and runs it through the embedded
`interp` interpreter. **There is no dependency on a system shell** — this is what makes taskctl
cross-platform. Exit codes surface via `IsExitStatus`.

**Output** — `internal/output`. `TaskOutput` wraps a task's stdout/stderr with a
`DecoratedOutputWriter` decorator chosen by format: `raw`, `prefixed`, or `cockpit` (live multi-task
dashboard). Interactive tasks force `raw`. The `prefixed` decorator renders through `internal/tui`
(palette + colorprofile writer). The `cockpit` dashboard is a `bubbletea` program implemented here in
`internal/output/cockpit.go` — it lives with its sole consumer rather than in `tui`, borrowing only the
`tui` palette; keeping the `DecoratedOutputWriter` bridge here avoids an `output`↔`tui` import cycle.

**Contexts** — `runner/context.go`. An `ExecutionContext` can wrap commands in an executable (e.g.
`docker`, `bash -c`), set a dir/env, and define `up`/`down`/`before`/`after` lifecycle hooks. Contexts
touched during a run are cleaned up (`Down`) on `Finish`.

**Watchers** — `internal/watch` uses `fsnotify` + `bmatcuk/doublestar` globs to re-trigger tasks on
file changes (`taskctl watch ...`).

### internal/ helpers

`internal/` packages are private, single-purpose helpers: `fsutil` (path/file checks), `envutil`
(env map ↔ `KEY=VAL` slice conversion), `iox` (`iox.Close` deferred-close helper), `tmpl` (template
rendering), `tui` (shared terminal-UI primitives: color palette, TTY detection, styled-print helpers,
and huh-based prompts), and `output` (task-output decorators, including the `bubbletea` cockpit
dashboard). Keep these focused; don't reintroduce a grab-bag utils package. `huh`/`lipgloss`/
`colorprofile` live in `tui`; `bubbletea` lives in `output` with the cockpit (its only consumer).

## Conventions

- Errors are wrapped with `%w`; check with `errors.Is`/`errors.As`. Deferred `Close()` errors must be
  handled (errcheck is enforced) — use `iox.Close`.
- Prefer stdlib generics helpers already adopted here: `maps.Keys`+`slices.Collect`, `slices.*`,
  `strings.Cut`.
- Logging is `log/slog` (level set from `--debug`/`TASKCTL_DEBUG`).
- Comments are concise — one line by default. Always comment exported (public) symbols; the reusable
  core packages (`runner`, `scheduler`, `task`, `executor`, `variables`) carry doc comments on every
  exported symbol — maintain them. For unexported methods and variables, comment only when genuinely
  needed — when what the code does, or why a variable exists, is not obvious from the code itself.
  Don't restate the obvious.
- Every package has table-style `_test.go` tests alongside; `cmd/` and `internal/config/` use
  `testdata/` fixtures.
- Do not use branch prefixes (`feat/`, `fix/`, `chore/`, …) — use plain branch names
  (e.g. `pipeline-task-variables`, not `fix/pipeline-task-variables`). Commit messages still use
  conventional prefixes (`feat:`, `fix:`, …).

## Development process

- **Plan first, then implement.** For any non-trivial change, agree on an approach before touching
  code — surface assumptions, edge cases, and trade-offs up front. Don't start editing while the
  design is still open.
- **Run tests and linters once, at the end** — not after every intermediate stage. Make the full set
  of changes, then verify by dogfooding: `go run . --output json --no-input prepare` (tidy, test,
  format, lint, completers), plus `go test -race ./...` for the race gate CI enforces (no task
  covers it). Check the tree is clean afterwards — `prepare` rewrites formatting and generated
  files, and an unexpected diff means something drifted.
- **Sync the docs before every PR.** Before opening or updating a pull request, invoke the
  `docs-sync` agent (`.claude/agents/docs-sync.md`) so `README.md` and `docs/` reflect the change
  set. Don't hand-wave this — the agent reconciles docs against the actual diff and fixes drift.

## Agent skill (single source of truth)

The taskctl agent skill lives once at `.agents/skills/taskctl/SKILL.md`. It is embedded into the
binary from `main.go` (go:embed, then injected into `cmd` via `SetSkillTemplate`) and shipped by
`taskctl skill install`; `.claude/skills/taskctl` symlinks to it so this repo's own agents use the
same copy. Edit only the canonical file — never a copy.
