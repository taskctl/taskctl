---
name: taskctl
description: Run project tasks and pipelines with taskctl. Use when the project contains tasks.yaml or taskctl.yaml, or when asked to build/test/deploy via project task definitions.
---

# Using taskctl

taskctl is a concurrent task runner. Tasks and pipelines are defined in the project config (tasks.yaml / taskctl.yaml), but do NOT parse those files — use the machine-readable CLI instead.

## Discover what exists

```bash
# List every task, pipeline, context and watcher as one JSON document
taskctl --output json list
```

Returns `{schema_version, tasks, pipelines, contexts, watchers}`. Task entries carry `name`, `description`, `context`; pipeline entries carry `name` and `stages`.

## Inspect before running

```bash
# Full resolved detail for one task or pipeline
taskctl --output json show <task-or-pipeline>
```

Tasks: resolved `commands`, `env`, `variables`, `dir`, `timeout_seconds`, `allow_failure`, `condition`. Pipelines: `stages` with `depends_on` edges (the execution DAG); a stage carries either `task` (the task it runs) or `pipeline` (a nested sub-pipeline).

## Execute

```bash
# Run a task or pipeline; <target> is passed directly, no `run` keyword
taskctl --output json --no-input <target>
```

Stdout is an NDJSON event stream — one JSON object per line:

| event | key fields |
|---|---|
| run_started | schema_version, targets |
| task_started | task |
| task_output | task, stream (stdout/stderr), data (one line) |
| task_finished | task, status (done/failed/skipped), exit_code, duration_ms, error |
| run_finished | status (done/failed), duration_ms, tasks[] (per-task status: done/failed/skipped/canceled), error (present on failure) |

`run_finished.status` is the source of truth for success. Exit code is 0 on success, non-zero on failure. taskctl's own diagnostics go to stderr.

## Rules

- Always pass `--output json --no-input`.
- Never invoke interactive commands (`taskctl` with no arguments opens a selector when in a terminal).
- Prefer running a pipeline over hand-sequencing its tasks — taskctl handles ordering and concurrency.
