# CLI UX overhaul — session summary

Planning session on **2026-07-18** (branch `context-variables`). Goal: fix a cluster of CLI defects, migrate the CLI framework to cobra, and land several UX improvements — planned as three sequential streams. No code was written this session; this is the plan of record.

## What we're solving

The user reported seven CLI issues:

1. Rename the `cockpit` output format to `default` and make it the default; update docs.
2. On exit the dashboard output disappears — keep it on screen with a richer summary.
3. Remove the `--cockpit` shortcut for `--output=cockpit`.
4. `--summary` appears ignored in cockpit mode.
5. `taskctl list --output json` doesn't work.
6. `--raw` is ignored, errors with `unknown task or pipeline "--raw"`.
7. Audit that other global flags and commands work in any position.

Plus a framework change: **migrate from `urfave/cli/v2` to `spf13/cobra` (latest, ~v1.10.x)**.

## Root-cause finding

Issues **#4, #5, #6, #7 are one bug**: urfave/cli/v2 parses with stdlib `flag`, which **stops at the first positional argument** and has **no persistent-flag inheritance**. Verified in `urfave/cli/v2@v2.27.7/parse.go` — `parseIter` calls `set.Parse`, which halts at the first non-flag token. So any flag placed *after* a target (`list --output json`, `build --raw`, `<task> --summary=false`) is either rejected as undefined or swallowed as a target name (the `--raw` crash originates at `cmd/run.go:142`).

Rather than hoist args around this, cobra dissolves the whole class: **pflag intersperses flags and positionals**, and **persistent flags are inherited by every subcommand**. Cobra also brings native shell completions, did-you-mean, and grouped help for free.

Two other corrections surfaced during exploration:

- **#2** ("output disappears"): the summary *does* print after the dashboard exits — the real gap is that the dashboard only shows spinners + "Finished" lines (never command stdout), so the thin summary is a weak final artifact. Fix = enrich the summary (Stream B) + feed live output into the dashboard (Stream C).
- **#4** ("--summary ignored in cockpit"): the summary is not actually suppressed in cockpit mode (only JSON suppresses it). The real defect was flag placement — `<task> --summary=false` was swallowed as a target. Stream A's parsing fix resolves it.

## User decisions captured

- **One combined effort, split into three sequential streams**; improvements that fall out of the cobra rewrite go in the migration stream.
- **Command layout: one directory per command** (`cmd/run/`, `cmd/list/`, …). Go's package-per-directory rule forces shared state into a shared package → `internal/cmdutil` with a `State` struct passed to each command constructor.
- **Summary content**: counts header + per-task status/duration + start times + exit codes + captured-output size + failed-task log tails — everything we know — and **extended to single-task runs** (today it's pipeline-only).
- **Non-TTY fallback** for the `default` format = `prefixed` (unchanged rule).
- **UX extras** (all accepted): native shell completions, did-you-mean target suggestions, grouped + styled help, live output in the dashboard.
- **Plans saved to `docs/plans/`, one file per stream.**
- Standing prefs: never commit automatically; comments only for non-obvious *why*; no branch prefixes; docs-sync agent before every PR; each non-trivial stream ends with a code-review + simplify pass.

## The three streams

| File | Stream | Scope |
| --- | --- | --- |
| `cli-cobra-migration.md` | **A** | Cobra migration, per-command package split (`cmd/*/` + `internal/cmdutil`), `cockpit`→`default` rename, `--cockpit` removal, persistent-flag fixes (#4–#7), native completions, did-you-mean, grouped help, tests + docs. |
| `cli-run-summary.md` | **B** | Rich persistent end-of-run summary — counts header, aligned per-stage lines, exit codes, output size, failed-task stderr tails; extended to single tasks. New `cmd/run/summary.go`. |
| `cli-live-dashboard.md` | **C** | Live command output in the `default` dashboard — per-task last-line rendering with partial-line buffering, width truncation, row cap. `internal/output/cockpit.go`. |

Streams are ordered: A (structural, unblocks everything) → B (makes the persistent summary worth reading) → C (live view that pairs with the summary). B and C are independent in design but both assume A's cobra files and the `default` format.

## Key facts pinned down (verified against source)

- Global flags & resolution: `cmd/cmd.go:80-133` (flags), `cmd/cmd.go:134-179` (`Before` hook); current default is `prefixed` (`cmd.go:165-166`); cockpit→prefixed non-TTY downgrade at `cmd.go:174-176`.
- `run` re-declares `--dry-run`/`--summary` locally (`cmd/run.go:31-42`) — duplication to delete.
- Format constants: `internal/output/output.go:11-16`; dispatch `output.go:45-64`. Dashboard: `internal/output/cockpit.go` (inline bubbletea, singleton, torn down via `output.Close()` ← `runner.TaskRunner.Finish()`, `runner/runner.go:207`).
- Summary today: `printSummary` `cmd/run.go:316-346`, called from `runPipeline` `run.go:152-174` after teardown; pipelines only.
- `--` handling relies on the literal `--` surviving in `c.Args()` (`run.go:179-192`, `300-314`); cobra strips it → use `cmd.ArgsLenAtDash()`.
- Completion: hand-rolled `cmd/completion.go`; artifacts in `autocomplete/`; regenerated by `update-completers` (`tasks.yaml:63-69`); dynamic names via app `BashComplete` (`cmd.go:69-79`). Cobra replaces this with a native `completion` subcommand + `ValidArgsFunction`.
- Doc touch-points: README L29, L204, L391-396, L511-517; `docs/example.yaml:163`; `tasks.yaml` update-completers L63-69; CLAUDE.md architecture paragraph.

## Status

Plans finalized and written to `docs/plans/`. Implementation not started — awaiting the user's go-ahead to execute Stream A first.
