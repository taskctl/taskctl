# Stream C — Live output in the dashboard

## Context

The `cockpit`→`default` dashboard (`internal/output/cockpit.go`) is an inline bubbletea program (no alt-screen), a process-wide singleton torn down via `output.Close()` ← `runner.TaskRunner.Finish()` (`runner/runner.go:207`). Today it renders only a spinner per running task plus a persistent `tea.Println` "✔/✗ Finished …" line when each task completes — it **never shows command output while a task runs**. This stream feeds each task's latest output line into the dashboard so users see live progress, not just spinners.

This is the last stream because it depends on the `default` format existing (Stream A) and pairs with the enriched summary (Stream B) that persists after the live view exits.

---

## Stream C — Live output in the dashboard

### C1. Feed output into the model — `internal/output/cockpit.go`

- New msg `type taskOutputMsg struct { name, line string }`.
- The cockpit decorator's `Write` (currently the dashboard ignores command output) parses the chunk, keeps the last non-empty line, and `prog.Send(taskOutputMsg{...})`. Buffer partial lines per writer (a `bytes.Buffer` carry) so mid-line chunks don't render garbage.
- Model gains `lastLine map[string]string` and `started map[string]time.Time`; `taskStartedMsg` records start, `taskFinishedMsg` deletes both entries (keep the existing `tea.Println` "✔/✗ Finished …" persistent lines).

### C2. Render

- `View()` becomes one row per running task: `spinner  name  (elapsed)` + a second faint line with the task's last output line, truncated to terminal width (`tea.WindowSizeMsg` → store width; truncate with `ansi.Truncate` from lipgloss's ansi package, already in the dep tree). Cap visible rows (e.g. 8) with a `… and N more` line so inline mode doesn't fight scrollback.
- Elapsed ticks: reuse the spinner's tick to re-render; no extra timer.

### C3. Tests

- Extend `internal/output` tests: model `Update` handling of `taskOutputMsg` (last-line wins, cleared on finish), `Write` partial-line carry, truncation, and the existing teardown behavior unchanged.

**Verification C**: TTY run of `prepare` — each running task shows its live last output line; lines clear when tasks finish; "Finished" lines + Stream B summary persist after exit; non-TTY run unaffected (prefixed).

---

## Cross-cutting (applies to all streams)

- Commit this doc with the stream's changes.
- Run tests/linters once at the end of the stream — `go run . --output json --no-input prepare` + `go test -race ./...`; tree must be clean afterwards.
- End with a main-model code review + simplify pass.
- Docs-sync agent before the PR. No branch prefixes. No commits unless explicitly requested.
