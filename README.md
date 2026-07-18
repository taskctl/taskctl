<p align="center">
<img width="400" src="https://raw.githubusercontent.com/taskctl/taskctl/main/docs/logo.png" alt="taskctl - developer's routine tasks automation toolkit" title="taskctl - developer's routine tasks automation toolkit" />
</p>

# taskctl - concurrent task runner, developer's routine tasks automation toolkit
[![pkg.go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat)](https://pkg.go.dev/github.com/taskctl/taskctl?tab=doc)
![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/taskctl/taskctl)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/taskctl/taskctl)
![GitHub closed issues](https://img.shields.io/github/issues-closed/taskctl/taskctl)
![GitHub issues](https://img.shields.io/github/issues/taskctl/taskctl)
![Licence](https://img.shields.io/github/license/taskctl/taskctl)

[![Tests](https://github.com/taskctl/taskctl/actions/workflows/pull-request-checks.yml/badge.svg)](https://github.com/taskctl/taskctl/actions/workflows/pull-request-checks.yml)
![GitHub top language](https://img.shields.io/github/languages/top/taskctl/taskctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/taskctl/taskctl)](https://goreportcard.com/report/github.com/taskctl/taskctl)
[![Test Coverage](https://codecov.io/gh/taskctl/taskctl/branch/main/graph/badge.svg)](https://codecov.io/gh/taskctl/taskctl)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat)](https://github.com/taskctl/taskctl/pulls)

A simple, modern alternative to GNU Make. *taskctl* is a concurrent task runner that allows you to design your routine tasks and development pipelines in a nice and neat way in a human-readable format (YAML, JSON or TOML). Given a pipeline (composed of tasks or other pipelines), it builds a graph that outlines the execution plan. Tasks may run concurrently or cascade. Besides pipelines, each single task can be started manually or triggered by the built-in filesystem watcher.

## Features
- human-readable configuration (YAML, JSON or TOML) with local or remote imports
- concurrent task execution with DAG-based pipelines: dependencies, conditions, allowed failures, graph visualization
- cross-platform: embedded shell interpreter, no dependency on a system shell
- AI-agent friendly: JSON discovery, NDJSON run events, non-interactive mode, installable agent skill
- customizable execution contexts (wrap commands in `docker`, `ssh`, any binary)
- templated commands with variables, task variations, and output piped between tasks
- integrated file watcher (live reload)
- output formats: raw, prefixed, live cockpit dashboard, or JSON event stream
- interactive task selector and shell autocomplete
- embeddable task runner for Go programs

```yaml
tasks:
  lint:
    command:
      - golangci-lint run
      - go vet ./...
  
  test:
    allow_failure: true
    command: go test ./...
        
  build:
    command: go build -o bin/app ./...
    env: 
      GOOS: linux
      GOARCH: amd64
    before: rm -rf bin/*

pipelines:
  release:
    - task: lint
    - task: test
    - task: build
      depends_on: [lint, test]
```
According to this plan, `lint` and `test` will run concurrently, and `build` will start only when both `lint` and `test` have finished.


[![asciicast](https://asciinema.org/a/326726.svg)](https://asciinema.org/a/326726)

## Contents
- [Getting started](#getting-started)
  - [Installation](#install)
  - [Usage](#usage)
- [taskctl for AI agents](#taskctl-for-ai-agents)
- [Configuration](#configuration)
    - [Global configuration](#global-configuration)
    - [Example](#example)
- [Tasks](#tasks)
    - [Pass CLI arguments to task](#pass-cli-arguments-to-task)
    - [Task's variations](#tasks-variations)
    - [Task's variables](#tasks-variables)
    - [Storing task's output](#storing-tasks-output)
    - [Conditional execution](#task-conditional-execution)
- [Pipelines](#pipelines)
- [Output formats](#taskctl-output-formats)
- [Filesystem watchers](#filesystem-watchers)
    - [Patterns](#patterns)
- [Contexts](#contexts)
- [CLI reference](#cli-reference)
- [Embeddable task runner](#embeddable-task-runner)
    - [Runner](#runner)
    - [Scheduler](#scheduler)
- [Autocomplete](#autocomplete)
- [How to contribute?](#how-to-contribute)
- [License](#license)

## Getting started
### Install
#### macOS
```
brew tap taskctl/taskctl
brew install taskctl
```
#### Linux
```
sudo wget https://github.com/taskctl/taskctl/releases/latest/download/taskctl_linux_amd64 -O /usr/local/bin/taskctl
sudo chmod +x /usr/local/bin/taskctl
```
#### deb/rpm:
Download the .deb or .rpm from the [releases](https://github.com/taskctl/taskctl/releases) page and install with `dpkg -i` and `rpm -i` respectively.

#### Windows
```
scoop bucket add taskctl https://github.com/taskctl/scoop-taskctl.git
scoop install taskctl
```
#### From sources
```
git clone https://github.com/taskctl/taskctl
cd taskctl
go build -o taskctl .
```
#### Docker images
Docker images are available on [Docker Hub](https://hub.docker.com/r/taskctl/taskctl) and [GitHub Container Registry](https://github.com/taskctl/taskctl/pkgs/container/taskctl) (`ghcr.io/taskctl/taskctl`).

### Usage
- `taskctl` - run the interactive task prompt
- `taskctl pipeline1` - run a single pipeline
- `taskctl task1` - run a single task
- `taskctl pipeline1 task1` - run one or more pipelines and/or tasks
- `taskctl watch watcher1 watcher2` - start one or more watchers

## taskctl for AI agents

taskctl has a machine-readable CLI surface designed for use by AI agents and other tooling: JSON discovery documents, an NDJSON event stream for runs, non-interactive execution, and an installable Claude Code skill.

### Discovering tasks and pipelines: `--output json list` / `show`

`taskctl --output json list` prints a single JSON document describing every task, pipeline, context and watcher in the config:

```json
{
  "schema_version": 1,
  "tasks": [
    {"name": "lint", "description": "", "context": ""}
  ],
  "pipelines": [
    {"name": "release", "stages": ["build", "lint", "test"]}
  ],
  "contexts": [],
  "watchers": []
}
```

`taskctl --output json show <name>` prints the full detail for a single task or pipeline:

```json
{
  "schema_version": 1,
  "task": {
    "name": "lint",
    "commands": ["golangci-lint run"],
    "env": {},
    "variables": {},
    "allow_failure": false
  }
}
```

Pipelines are shown as a name plus a list of stages (sorted by stage name for stability), each with its task (or `pipeline` for nested sub-pipelines), dependencies and conditions:

```json
{
  "schema_version": 1,
  "pipeline": {
    "name": "release",
    "stages": [
      {"name": "build", "task": "build", "depends_on": ["lint", "test"], "allow_failure": false}
    ]
  }
}
```

### Streaming run events: `--output json`

Running with `--output json` switches the run's output to newline-delimited JSON (NDJSON) — one event object per line — instead of human-oriented text. This makes it easy for an agent to parse progress and results without screen-scraping.

```
taskctl --output json --no-input <task-or-pipeline>
```

`<task-or-pipeline>` is passed directly, with no `run` keyword. The stream is:

| event | key fields |
|---|---|
| `run_started` | `schema_version`, `targets` |
| `task_started` | `task` |
| `task_output` | `task`, `stream` (`stdout`/`stderr`), `data` |
| `task_finished` | `task`, `status` (`done`/`failed`/`skipped`), `exit_code`, `duration_ms`, `error` (on failure) |
| `run_finished` | `status` (`done`/`failed`), `duration_ms`, `tasks` (array of `{task, status (done/failed/skipped/canceled), exit_code, duration_ms}`), `error` (on failure) |

### Non-interactive execution: `--no-input`

By default taskctl may prompt interactively (e.g. for confirmation or input tasks). Non-interactive mode disables all of that, and is enabled whenever either of the following is true:

- `--no-input` is passed, or the `TASKCTL_NO_INPUT` environment variable is set
- output format is `json`

A non-TTY stdin (e.g. a pipe or an agent harness) does *not* by itself enable non-interactive mode — prompts still run in accessible, line-based mode against the pipe. It only affects the no-target case: when you run `taskctl` with no task or pipeline, the interactive selector requires a TTY, so on a non-TTY stdin taskctl errors with guidance instead of blocking. Pass `--no-input` (or `--output json`) to suppress prompts explicitly.

Separately, `--cockpit` (the live full-screen dashboard) requires an interactive stdout; if stdout is not a TTY, taskctl automatically degrades cockpit output to `prefixed` output instead of failing.

### Claude Code skill: `taskctl skill install`

`taskctl skill install` writes a Claude Code skill (`SKILL.md`) that teaches an agent how to use taskctl's JSON surface, into `.claude/skills/taskctl/SKILL.md` in the current directory.

- `--global` installs into the user's home directory instead of the current directory.
- `--force` overwrites an existing installation.

## Configuration
*taskctl* uses a config file (`tasks.yaml` or `taskctl.yaml`) where your tasks and pipelines are stored. The config file includes the following sections:
- tasks
- pipelines
- watchers
- contexts
- variables

A config file may import other config files, directories or URLs.
```yaml
import:
- .tasks/database.yaml
- .tasks/lint/
- https://raw.githubusercontent.com/taskctl/taskctl/main/docs/example.yaml
```

### Example
Config file [example](https://github.com/taskctl/taskctl/blob/main/docs/example.yaml)

### Global configuration
*taskctl* has a global configuration stored in the ``$HOME/.taskctl/config.yaml`` file. It is handy for storing system-wide tasks, reusable contexts, defaults, etc.

## Tasks
A task is the foundation of *taskctl*. It describes one or more commands to run, their environment, executors and attributes such as the working directory, execution timeout, acceptance of failure, etc.
```yaml
tasks:
    lint:
        allow_failure: true
        command:
          - golint $(go list ./... | grep -v /vendor/)
          - go vet $(go list ./... | grep -v /vendor/)
          
    build:
        command: go build ./...
        env: 
          GOOS: linux
          GOARCH: amd64
        env_file: /data/.env
        after: rm -rf tmp/*
        variations:
          - GOARCH: amd64
          - GOARCH: arm
            GOARM: 7
```
A task definition takes the following parameters:
- `command` - one or more commands to run
- `description` - human-readable description, shown by `taskctl list` and `taskctl show`
- `variations` - list of variations (env variables) to apply to command
- `context` - execution context's name
- `env` - environment variables. All existing environment variables will be passed automatically
- `env_file` - env file in `k=v` format to read variables from
- `dir` - working directory. Current working directory by default
- `timeout` - command execution timeout (default: none)
- `allow_failure` - if set to `true`, failed commands will not interrupt execution (default: `false`)
- `after` - command that will be executed after the task completes
- `before` - command that will be executed before the task starts
- `exportAs` - env variable name to store the task's output (default: `TASK_NAME_OUTPUT`, where `TASK_NAME` is the actual task's name)
- `condition` - condition to check before running task
- `variables` - task's variables
- `interactive` - if `true` provides STDIN to commands (default: `false`)

### Tasks variables
Each task, stage and context has variables that are used to render a task's fields - `command`, `dir`, `before`, `after`. Along with the globally predefined ones, variables can be set in a task's definition. You can use those variables according to the `text/template` [documentation](https://pkg.go.dev/text/template).

Variables layer by precedence, last wins: global < context < task. So a variable declared under a context's `variables:` is available in the `command`, `dir`, `before` and `after` of any task using that context, and a task-level variable of the same name overrides it. (A task's `condition:` sees global variables only.)

Predefined variables are:
- `.Root` - root config file directory
- `.Dir` - config file directory
- `.TempDir` - system's temporary directory
- `.Args` - provided arguments as a string
- `.ArgsList` - array of provided arguments
- `.Task.Name` - current task's name
- `.Context.Name` - current task's execution context's name
- `.Stage.Name` - current stage's name
- `.Output` - previous command's output
- `.Tasks.Task1.Output` - `task1` last command output

Variables can be used inside task definition. For example:
```yaml
tasks:
    task1:
        dir: "{{ .Root }}/some-dir"
        command:
          - echo "My name is {{ .Task.Name }}"
          - echo {{ .Output }} # My name is task1
          - echo "Sleep for {{ .sleep }} seconds"
          - sleep {{ .sleep | default 10 }}
          - sleep {{ .sleep }}
        variables:
          sleep: 3
```

### Pass CLI arguments to task
Any command line arguments succeeding `--` are passed to each task via the `.Args` and `.ArgsList` variables or the `ARGS` environment variable.

Given this definition:
```yaml
lint1:
  command: go lint {{.Args}}

lint2:
  command: go lint {{index .ArgsList 1}}
```
the resulting command is:
```
$ taskctl lint1 -- package.go
# go lint package.go

$ taskctl lint2 -- package.go main.go
# go lint main.go
```

### Storing task's output
A task's output is automatically stored in a variable named ``.Tasks.TaskName.Output``, where `TaskName` is the actual task's name. It is also stored in the `TASK_NAME_OUTPUT` environment variable, whose name can be changed with the task's `exportAs` parameter. Those variables are available to all dependent stages.

### Tasks variations
A task may run in one or more variations. Variations allow you to reuse a task with different env variables:
```yaml
tasks:
  build:
    command:
      - GOOS=${GOOS} GOARCH=amd64 go build -o bin/taskctl_${GOOS} ./cmd/taskctl
    env:
      GOFLAGS: -ldflags=-s -ldflags=-w
    variations:
      - GOOS: linux
      - GOOS: darwin
      - GOOS: windows
```
This config will run the build 3 times, each with a different `GOOS`.

### Task conditional execution
The following task will run only when there are any changes that are staged but not committed:
```yaml
tasks:
  build:
    command:
      - ...build...
    condition: git diff --exit-code
```

## Pipelines
A pipeline is a set of stages (tasks or other pipelines) to be executed in a certain order. Stages may be executed in parallel or one-by-one. A stage may override the task's environment, variables, etc.

This pipeline:
```yaml
pipelines:
    pipeline1:
        - task: start task
        - task: task A
          depends_on: "start task"
        - task: task B
          depends_on: "start task"
        - task: task C
          depends_on: "start task"
        - task: task D
          depends_on: "task C"
        - task: task E
          depends_on: ["task A", "task B", "task D"]
        - task: finish
          depends_on: ["task E"]    
```
will result in an execution plan like this:
![execution plan](https://raw.githubusercontent.com/taskctl/taskctl/main/docs/pipeline.svg)

A stage definition takes the following parameters:
- `name` - stage name. If not set, the referenced task or pipeline name will be used.
- `task` - task to execute on this stage
- `pipeline` - pipeline to execute on this stage
- `env` - environment variables. All existing environment variables will be passed automatically
- `env_file` - file with env variables in `k=v` format to read variables from
- `dir` - working directory override for the task run in this stage
- `depends_on` - names of the stages this stage depends on. This stage will be started only after the referenced stages have completed.
- `allow_failure` - if `true`, a failing stage will not interrupt pipeline execution. ``false`` by default
- `condition` - condition to check before running stage
- `variables` - stage's variables

## Taskctl output formats
Taskctl has several output formats:
- `raw` - prints raw commands output
- `prefixed` - strips ANSI escape sequences where possible, prefixes command output with task's name
- `cockpit` - tasks dashboard
- `json` - newline-delimited JSON event stream for machine consumption (see [taskctl for AI agents](#taskctl-for-ai-agents))

## Filesystem watchers
A watcher watches for changes in files selected by the provided patterns and triggers the task any time an event occurs.
```yaml
watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"] # Files to watch
    exclude: ["pkg/excluded.go", "pkg/excluded-dir/*"] # Exclude patterns
    events: [create, write, remove, rename, chmod] # Filesystem events to listen to
    task: task1 # Task to run when event occurs
```

A watcher definition takes the following parameters:
- `watch` - patterns of files to watch
- `exclude` - patterns of files to exclude
- `events` - filesystem events to listen to (`create`, `write`, `remove`, `rename`, `chmod`)
- `task` - task to run when an event occurs
- `variables` - watcher's variables, passed to the task

### Patterns
Thanks to [doublestar](https://github.com/bmatcuk/doublestar) *taskctl* supports the following special terms within include and exclude patterns:

| Special Terms | Meaning |
|---|---|
| `*` | matches any sequence of non-path-separators |
| `**` | matches any sequence of characters, including path separators |
| `?` | matches any single non-path-separator character |
| `[class]` | matches any single non-path-separator character against a class of characters ([details](https://github.com/bmatcuk/doublestar/blob/master/README.md#character-classes)) |
| `{alt1,...}` | matches a sequence of characters if one of the comma-separated alternatives matches |

Any character with a special meaning can be escaped with a backslash (`\`).

## Contexts
Contexts allow you to set up the execution environment, variables, the binary that will run your task, up/down commands, etc.
```yaml
contexts:
  local:
    executable:
      bin: /bin/zsh
      args:
        - -c
    env:
      VAR_NAME: VAR_VALUE
    variables:
      sleep: 10
    quote: "'" # will quote command with provided symbol: "/bin/zsh -c 'echo 1'"
    before: echo "I'm local context!"
    after: echo "Have a nice day!"
```

A context definition takes the following parameters:
- `dir` - working directory. Also the base for a relative `env_file` path
- `executable` - binary (`bin`) and its arguments (`args`) that will run the task's commands
- `quote` - symbol to quote commands with when passing them to the executable
- `env` - environment variables
- `env_file` - file with env variables in `k=v` format to read variables from
- `variables` - context's variables
- `up`, `down`, `before`, `after` - lifecycle hooks (see below)

A context has lifecycle hooks: `up` and `down` run once per taskctl run - `up` before the context's first usage, `down` during cleanup when the run finishes. `before` and `after` run every time around each task that uses the context.
```yaml
context:
    docker-compose:
      executable:
        bin: docker-compose
        args: ["exec", "api"]
      up: docker-compose up -d api
      down: docker-compose down api

    local:
      after: rm -rf var/*
```

### Docker context
```yaml
  alpine:
    executable:
      bin: /usr/local/bin/docker
      args:
        - run
        - --rm
        - alpine:latest
    env:
      DOCKER_HOST: "tcp://0.0.0.0:2375"
    before: echo "SOME COMMAND TO RUN BEFORE TASK"
    after: echo "SOME COMMAND TO RUN WHEN TASK FINISHED SUCCESSFULLY"

tasks:
  mysql-task:
    context: alpine
    command: uname -a
```

## CLI reference

### Commands

| command | description |
|---|---|
| `taskctl [target...]` (or `taskctl run [target...]`) | run one or more pipelines and/or tasks; with no target, opens the interactive selector |
| `taskctl init` | create a sample config file in the current (or `--dir`) directory |
| `taskctl list` | list all tasks, pipelines and watchers; `list tasks`, `list pipelines`, `list watchers` narrow the output |
| `taskctl show <name>` | show a task's or pipeline's details |
| `taskctl watch <watcher...>` | start one or more filesystem watchers |
| `taskctl graph [pipeline]` (alias `g`) | visualize a pipeline's execution graph in DOT format (e.g. `taskctl graph release \| dot -Tsvg > graph.svg`); `--lr` orients it left-to-right |
| `taskctl validate` | validate the config file |
| `taskctl completion <shell>` | generate a completion script for `bash` or `zsh` |
| `taskctl skill install` | install the AI agent skill (see [taskctl for AI agents](#taskctl-for-ai-agents)) |

### Global flags

| flag | env variable | description |
|---|---|---|
| `-c, --config <file>` | `TASKCTL_CONFIG_FILE` | config file to use (default: `tasks.yaml` or `taskctl.yaml`) |
| `-o, --output <format>` | `TASKCTL_OUTPUT_FORMAT` | output format: `raw`, `prefixed`, `cockpit` or `json` |
| `-r, --raw` | | shortcut for `--output=raw` |
| `--cockpit` | | shortcut for `--output=cockpit` |
| `-q, --quiet` | | quiet mode |
| `--set <name=value>` | | set a global variable value (repeatable) |
| `--dry-run` | | resolve and print commands without executing them |
| `-s, --summary` | | show a run summary; on by default in human output modes, off with `--quiet` or in `raw` mode (unless opted in via config), never in `json`. An explicit flag wins over these defaults |
| `--no-input` | `TASKCTL_NO_INPUT` | disable interactive prompts |
| `-d, --debug` | `TASKCTL_DEBUG` | enable debug output |

## Embeddable task runner
*taskctl* may be embedded into any Go program. Additional information may be found on taskctl's [pkg.go.dev](https://pkg.go.dev/github.com/taskctl/taskctl?tab=overview) page.

### Runner
```go
t := task.FromCommands("go fmt ./...", "go build ./...")
r, err := NewTaskRunner()
if err != nil {
    return
}
err  = r.Run(t)
if err != nil {
    fmt.Println(err, t.ExitCode, t.ErrorMessage())
}
fmt.Println(t.Output())
```

### Scheduler
```go
format := task.FromCommands("go fmt ./...")
build := task.FromCommands("go build ./...")
r, _ := runner.NewTaskRunner()
s := NewScheduler(r)

graph, err := NewExecutionGraph(
    &Stage{Name: "format", Task: format},
    &Stage{Name: "build", Task: build, DependsOn: []string{"format"}},
)
if err != nil {
    return
}

err = s.Schedule(graph)
if err != nil {
    fmt.Println(err)
}
```

## Autocomplete
### Bash
Add to  ~/.bashrc or ~/.profile
```
. <(taskctl completion bash)
```

### ZSH
Add to  ~/.zshrc
```
. <(taskctl completion zsh)
```
## How to contribute?
Feel free to contribute in any way you want. Share ideas, submit issues, create pull requests. You can start by improving this [README.md](https://github.com/taskctl/taskctl/blob/main/README.md) or suggesting new [features](https://github.com/taskctl/taskctl/issues). Thank you!

## License
This project is licensed under the GNU GPLv3 - see the [LICENSE.md](LICENSE.md) file for details

## Authors
 - Yevhen Terentiev - [trntv](https://github.com/trntv)
See also the list of [contributors](https://github.com/taskctl/taskctl/contributors) who participated in this project.
