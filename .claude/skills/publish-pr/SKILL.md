---
name: publish-pr
description: Publish the current branch and open (or update) a GitHub pull request for the taskctl repo
disable-model-invocation: true
---

# publish-pr

Publish the current branch and create — or update — a GitHub pull request, using this repo's conventions and PR-description templates.

## Steps

1. **Pre-publish gate** (this repo requires it before every PR):
   - Run the `prepare` task and confirm it is clean: `go run . --output json --no-input prepare`, then check the tree is clean (`prepare` rewrites formatting/generated files — an unexpected diff means something drifted, fix it before continuing).
   - Run the `docs-sync` agent so `README.md` and `docs/` reflect the change set.
2. **Inspect git state:**
   - `git status --short --branch`
   - `git diff --stat`
   - `git log --oneline --decorate -10`
   - `git diff main...HEAD --stat`
3. **Commit any uncommitted relevant changes:**
   - Stage only the relevant files; write a concise commit matching repo style (conventional prefixes: `feat:`, `fix:`, `chore:`, …).
   - Do not amend unless explicitly requested.
4. **Push** the current branch with upstream: `git push -u origin HEAD`. Never force-push unless explicitly requested.
5. **Pick the template** from signals (see below).
6. **Compose title and body** (see templates and writing rules) from the whole branch diff (`main...HEAD`) and chat history — summarize *all* changes, not just the latest commit.
7. **Create or update the PR** with the GitHub CLI:
   - Check whether a PR already exists for the branch (`gh pr view --json url,title` or `gh pr list --head <branch>`).
   - If it exists — update the description to reflect the current change set.
   - If not — `gh pr create` and pass the body via a HEREDOC.
   - If the CLI cannot be used, fall back to the GitHub MCP tool.
8. Return the PR URL.

## Choosing the template

Detect the change type from signals, in order:

- **Conventional-commit prefixes** in `git log` (`fix:` → bug fix; `feat:` → feature).
- **Diff shape:** a small targeted change correcting behavior → bug fix; new capability / new files / new commands → feature.

Branch names carry no prefix in this repo (see Default assumptions), so don't rely on them. If the signals conflict or are absent, **ask** which template to use before writing the body.

## Templates

Every section except **Overview** is optional — include it only when it carries real content. Never add a section just to fill the shape; an empty or padded section is worse than an absent one.

### Feature

```md
## Overview

- High-level outcome: what this adds and for whom.

## Goal

- Why this work exists; the user or product problem it addresses.

## Decisions

- Key implementation choices and trade-offs; scope boundaries.

## Architecture

- Only when the change introduces or reshapes structure worth explaining (new packages, data flow, extension points). Skip for self-contained features.
```

### Bug fix

```md
## Overview

- What was broken and the observable symptom.

## Root cause

- Why it happened — the actual defect, not just the symptom.

## Decisions

- Notable choices in the fix; alternatives considered; scope boundaries.

## Fix

- What changed to resolve it. Skip when the Overview + Root cause already make the fix self-evident.
```

## Writing rules

- Keep the title concise and outcome-focused; use the matching conventional prefix (`feat: …`, `fix: …`).
- Prefer intent over implementation detail. Do not turn the body into a changelog or restate code changes already visible in the diff.
- Call out product or architecture decisions when they matter.

## Default assumptions

- Base branch is `main`.
- **Do not use branch prefixes** (`feat/`, `fix/`, `chore/`, …) — use plain branch names (e.g. `pipeline-task-variables`, not `fix/pipeline-task-variables`). No GitHub issues keys.
- Use the GitHub CLI for PR creation; return the PR URL.
