# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What this is

`taskctl` is a concurrent task runner / Make alternative. Tasks and pipelines are declared in a human-readable config (`tasks.yaml`/`taskctl.yaml`, also JSON/TOML). It is a CLI application; the `runner`, `scheduler`, `task`, `executor`, and `variables` packages hold the reusable core, while CLI-only support (interactive prompts and output rendering) lives under `internal/`. The exported API of those five core packages is the embedding boundary and is kept exported-minimal — prefer unexported for anything only the package itself needs.

## General

These guidelines bias toward caution over speed; for trivial tasks, use judgment.

- **Think before coding.** State assumptions explicitly; if multiple interpretations exist, present them — don't pick silently. If something is unclear, stop and ask. If a simpler approach exists, say so — push back when warranted.
- **Simplicity first.** Minimum code that solves the problem, nothing speculative: no features beyond what was asked, no abstractions for single-use code, no unrequested configurability, no error handling for impossible scenarios. If 200 lines could be 50, rewrite.
- **Surgical changes.** Touch only what the request requires: don't improve adjacent code, refactor what isn't broken, or delete pre-existing dead code (mention it instead). Match existing style. Remove the orphans your own changes create — nothing else. Every changed line should trace directly to the user's request.

## Commands

There is no Makefile. The repo **dogfoods itself**: routine work (test, lint, format, completers) runs through taskctl's own task definitions in `tasks.yaml`, driven from the current source — so every dev loop also exercises the tool being changed. Use the machine-readable interface described by the bundled taskctl skill (`.agents/skills/taskctl/SKILL.md`); don't parse `tasks.yaml` or hand-sequence its commands in bash.

```bash
go run . --output json list                       # discover tasks/pipelines (always current source)
go run . --output json show <task-or-pipeline>    # inspect resolved commands and the stage DAG
go run . --output json --no-input <target>        # run; run_finished.status is the source of truth
```

`go run .` is the default invocation — it can never run a stale binary. For repeated runs, build once and use the binary (`go run . --output json --no-input build-host`, then `./bin/taskctl ...`); rebuild after changing source.

Common targets: `prepare` (tidy → test → format → lint → completers → docs; run before wrapping up), `test`, `golangci-lint`, `fmt` (gofmt + goimports via `golangci-lint fmt`), `fixcs`, `build-host` (host binary for dogfooding), `build` (cross-platform release binaries), `update-completers`, `update-docs` (regenerates the `docs/cli/` Markdown reference tree via the dev-only `tools/gendocs` generator). `list` gives the authoritative set.

Raw Go commands — fallback for cases with no matching task, or to mirror CI exactly:

```bash
go test -race ./...                # race detector (CI runs -v -race; no task for this)
go test -run TestName ./runner/    # single test in one package
golangci-lint run                  # lint directly (golangci-lint v2; config in .golangci.yaml)
golangci-lint fmt                  # format Go sources (gofmt + goimports; same config)
go build -o bin/taskctl .          # host build without taskctl (same as build-host task)
```

CI (`.github/workflows/pull-request-checks.yml`) gates PRs on `golangci-lint` + `go test -v -race ./...`. Go version is pinned in `go.mod` (currently 1.26). Release is handled by GoReleaser on tag push.

## Architecture

Execution flows through two layers — a pipeline DAG on top, single-task compilation underneath.

**Entry** — `main.go` → `cmd.Run(version)` builds a `spf13/cobra` command tree (`cmd/root.go`, `NewRootCommand`). Subcommands live one-per-file in `cmd/*.go` (run, init, list, show, watch, graph, validate, skill), all in `package cmd`; each `newXCommand(cfg)` constructor closes over the shared `*config.Config` and loader that `NewRootCommand` creates, so there are no mutable package globals. Global flags are cobra persistent flags on the root, inherited by every subcommand and resolved in `PersistentPreRunE`; env-var fallbacks are wired by the `bindEnv` helper (pflag has no built-in env support). Completion is cobra-native (`completion` subcommand + `ValidArgsFunction` for dynamic task/pipeline names); there is no hand-rolled completion command. A bare invocation with no target opens an interactive `huh` selector (via `internal/tui`). `Run` uses `signal.NotifyContext` to turn SIGINT/SIGTERM into a context cancel that tears down in-flight tasks.

**Config** — `internal/config`. `Loader` (`loader.go`) reads YAML/JSON/TOML, resolves `import:` entries (local files, directories, or remote URLs), and merges them with `dario.cat/mergo`. Raw maps are decoded into typed structs via `go-viper/mapstructure/v2`. The result is a `config.Config` holding `Tasks`, `Pipelines` (already built into `scheduler.ExecutionGraph`s), `Contexts`, `Watchers`, and a `Variables` container.

**Pipelines / scheduling** — `scheduler`. A pipeline is an `ExecutionGraph`: a DAG whose nodes are `Stage`s and edges are `depends_on` relationships (cycle detection in `graph.go`). `Scheduler.Schedule` polls the graph on a 50ms tick; any `StatusWaiting` stage whose deps are all `Done`/`Skipped` is launched in its own goroutine, giving concurrent execution while respecting dependencies. Stages can nest sub-pipelines. `AllowFailure` and per-stage `Condition` gate propagation.

**Single task** — `runner.TaskRunner.Run` (`runner/runner.go`) is the core. For each task it: resolves the `ExecutionContext` (running `Up`/`Before` hooks), merges env + variables, checks the task `condition`, runs `before` commands, then calls `taskCompiler.compileTask`.

**Compilation** — `runner/compiler.go`. `compileTask` renders variable templates and expands `variations` into a **linked list of `executor.Job`s** (`job.Next`). Each command becomes one job; a task with N commands × M variations produces N×M chained jobs. Output of one command is fed to the next as the `Output` variable.

**Execution** — `executor`. `DefaultExecutor.Execute` renders the command template (`internal/tmpl`, Go `text/template`), parses it with `mvdan.cc/sh/v3/syntax`, and runs it through the embedded `interp` interpreter. **There is no dependency on a system shell** — this is what makes taskctl cross-platform. Exit codes surface via `IsExitStatus`.

**Output** — `internal/output`. `TaskOutput` wraps a task's stdout/stderr with a `DecoratedOutputWriter` decorator chosen by format: `raw`, `prefixed`, `default` (the live multi-task dashboard), or `json`. `default` is the default format on a TTY and downgrades to `prefixed` on a non-TTY stdout. Interactive tasks force `raw`. The `prefixed` decorator renders through `internal/tui` (palette + colorprofile writer). The `default` dashboard is a `bubbletea` program implemented here in `internal/output/dashboard.go` — it lives with its sole consumer rather than in `tui`, borrowing only the `tui` palette; keeping the `DecoratedOutputWriter` bridge here avoids an `output`↔`tui` import cycle.

**Contexts** — `runner/context.go`. An `ExecutionContext` can wrap commands in an executable (e.g. `docker`, `bash -c`), set a dir/env, and define `up`/`down`/`before`/`after` lifecycle hooks. Contexts touched during a run are cleaned up (`Down`) on `Finish`.

**Watchers** — `internal/watch` uses `fsnotify` + `bmatcuk/doublestar` globs to re-trigger tasks on file changes (`taskctl watch ...`).

### internal/ helpers

`internal/` packages are private, single-purpose helpers: `fsutil` (path/file checks), `envutil` (env map ↔ `KEY=VAL` slice conversion), `iox` (`iox.Close` deferred-close helper), `tmpl` (template rendering), `tui` (shared terminal-UI primitives: color palette, TTY detection, styled-print helpers, and huh-based prompts), and `output` (task-output decorators, including the `bubbletea` dashboard). Keep these focused; don't reintroduce a grab-bag utils package. `huh`/`lipgloss`/`colorprofile` live in `tui`; `bubbletea` lives in `output` with the dashboard (its only consumer).

## Conventions

- Errors are wrapped with `%w`; check with `errors.Is`/`errors.As`. Deferred `Close()` errors must be handled (errcheck is enforced) — use `iox.Close`.
- Prefer stdlib generics helpers already adopted here: `maps.Keys`+`slices.Collect`, `slices.*`, `strings.Cut`.
- Logging is `log/slog` (level set from `--debug`/`TASKCTL_DEBUG`).
- Package-level `const`s and `var`s live at the top of the file (after the imports), not next to the function that first uses them — a reader scanning a file finds all its declarations in one place.
- Resolve a naming collision by renaming the more local, less-recognizable identifier — not the more canonical one. In particular, when a local `const`/`var`/type clashes with a package you need to import, rename the local identifier and import the package under its real name; do not alias the import (e.g. rename a local `ansi` regexp to `ansiPattern` and import `.../x/ansi` as `ansi`, never `xansi`). A package's canonical name is what readers recognize; an alias hides it.
- Comments are rare. Default to zero. Add one only when it captures something the code cannot: a non-obvious invariant, a workaround, or the reason a choice was made over the obvious alternative. A comment that restates the signature or name (`// Run executes the task`) is a defect — delete it, exported or not. The Go "every exported symbol gets a doc comment" convention does **not** apply in this repo: outside the reusable core packages (`runner`, `scheduler`, `task`, `executor`, `variables`), an exported symbol gets **zero** comment unless it earns one by the rule above — being exported is not a reason. Inside those core packages an exported symbol does need a doc comment, but only the one sentence (or few if critical) a reader can't get from the name/signature alone (error conditions, ordering guarantees, side effects) — never a restatement. Do not use GitHub issue IDs in comments.
- Every package has table-style `_test.go` tests alongside; `cmd/` and `internal/config/` use `testdata/` fixtures.
- Do not use branch prefixes (`feat/`, `fix/`, `chore/`, …) — use plain branch names (e.g. `pipeline-task-variables`, not `fix/pipeline-task-variables`). Commit messages still use conventional prefixes (`feat:`, `fix:`, …).
- Do not use manual line breaks inside Markdown paragraphs or list items (`AGENTS.md`, `SKILL.md`, etc.) — write each paragraph or list item as a single line and let the renderer soft-wrap it. Reason: hard-wrapped source lines diff noisily on small edits and read awkwardly once rendered at a different width. Code blocks and tables are exempt.
- Private (unexported) functions or variables always goes last, after public (exported) ones. Order is consts -> variables -> structs -> constructors -> methods -> unexported functions.

## Development process

- **Plan first, then implement.** For any non-trivial change, agree on an approach before touching code — surface assumptions, edge cases, and trade-offs up front. Don't start editing while the design is still open.
- **Run tests and linters once, at the end** — not after every intermediate stage. Make the full set of changes, then verify by dogfooding: `go run . --output json --no-input prepare` (tidy, test, format, lint, completers), plus `go test -race ./...` for the race gate CI enforces (no task covers it). Check the tree is clean afterwards — `prepare` rewrites formatting and generated files, and an unexpected diff means something drifted.
- **Sync the docs before every PR.** Before opening or updating a pull request, invoke the `docs-sync` agent (`.claude/agents/docs-sync.md`) so `README.md`, agent skill and `docs/` reflect the change set. Don't hand-wave this — the agent reconciles docs against the actual diff and fixes drift.

## Agent skill (single source of truth)

The taskctl agent skill lives once at `.agents/skills/taskctl/SKILL.md`. It is embedded into the binary and shipped by `taskctl skill install`; `.claude/skills/taskctl` symlinks to it so this repo's own agents use the same copy. Edit only the canonical file — never a copy.
