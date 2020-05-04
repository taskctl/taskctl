<p align="center">
<img width="400" src="https://raw.githubusercontent.com/taskctl/taskctl/master/docs/logo.png" alt="taskctl - developer's routine tasks automation toolkit" title="taskctl - developer's routine tasks automation toolkit" />
</p>

# taskctl - developer's routine tasks automation toolkit
![Tests](https://github.com/taskctl/taskctl/workflows/Test/badge.svg)
[![Requirements Status](https://requires.io/github/taskctl/taskctl/requirements.svg?branch=master)](https://requires.io/github/taskctl/taskctl/requirements/?branch=master)
![GitHub top language](https://img.shields.io/github/languages/top/taskctl/taskctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/taskctl/taskctl)](https://goreportcard.com/report/github.com/taskctl/taskctl)
[![Test Coverage](https://api.codeclimate.com/v1/badges/a99a88d28ad37a79dbf6/test_coverage)](https://codeclimate.com/github/codeclimate/codeclimate/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/a99a88d28ad37a79dbf6/maintainability)](https://codeclimate.com/github/codeclimate/codeclimate/maintainability)

![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/taskctl/taskctl)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/taskctl/taskctl)
![GitHub closed issues](https://img.shields.io/github/issues-closed/taskctl/taskctl)
![GitHub issues](https://img.shields.io/github/issues/taskctl/taskctl)
![Licence](https://img.shields.io/github/license/taskctl/taskctl)

Simple modern alternative to GNU Make. taskctl allows you to design you development workflow pipelines in nice and neat way in human-readable format (YAML, JSON or TOML). Each pipeline composed of tasks or other pipelines and allows them to run in parallel or one-by-one. 
Beside pipelines, each single task can be performed manually or triggered by built-in filesystem watcher.

## Features
- parallel tasks execution
- highly customizable pipelines configuration
- file watcher integrated with tasks and pipelines
- customizable contexts for each task
- human-readable configuration format (YAML, JSON or TOML)
- customizable output
- output capturing
- and many more...

[![asciicast](https://asciinema.org/a/304339.svg)](https://asciinema.org/a/304339)

## Contents  
- [Getting started](#getting-started)
  - [Installation](#install)
  - [Usage](#usage)
- [Configuration](#configuration)
    - [Global configuration](#global-configuration)
    - [Example](#example)
- [Tasks](#tasks)
    - [Task`s variations](#tasks-variations)
    - [Task`s variables](#tasks-variables)
    - [Storing task's output](#storing-tasks-output) 
    - [!!!Conditional execution](#task-conditional-execution) 
- [Pipelines](#pipelines)
    - [!!!Conditional execution](#conditional-execution) 
- [Filesystem watchers](#filesystem-watchers)
- [Contexts](#contexts)
- [FAQ](#faq)
  - [How does it differ from go-task/task?](#how-does-it-differ-from-go-tasktask)
- [Autocomplete](#autocomplete)
- [Similar projects](#similar-projects)
- [How to contribute?](#how-to-contribute)
- [License](#license)

## Getting started
### Install
#### MacOS
```
brew tap taskctl/taskctl
brew install taskctl
```

#### Linux
```
sudo wget https://github.com/taskctl/taskctl/releases/latest/download/taskctl_linux_amd64 -O /usr/local/bin/taskctl
sudo chmod +x /usr/local/bin/taskctl
```
#### From sources
```
go get -u github.com/taskctl/taskctl/cmd/taskctl
```

#### Install script
```
curl -sL https://raw.githubusercontent.com/taskctl/taskctl/master/install.sh | sh
```

### Usage
### Run pipeline
```
taskctl pipeline1 // single pipeline
taskctl pipeline1 pipeline2 // multiple pipelines
taskctl run pipeline1 // single pipeline, long form
```
### Run task
```
taskctl task1 // single task
taskctl task1 task2 // multiple tasks
taskctl run task task1 // multiple tasks, long form
```
### Start filesystem watcher
```
taskctl watch watcher1
```
### Set config file
```
taskctl -c tasks.yaml run lint
taskctl -c https://raw.githubusercontent.com/taskctl/taskctl/master/example/full.yaml run task4
```

## Configuration
taskctl uses config file (`tasks.yaml` or `taskctl.yaml`) to store tasks and pipelines. Config file includes following sections:
- tasks
- pipelines
- watchers
- contexts
- variables

Config file may import other config files, directories or URLs.
```yaml
import:
- .tasks/database.yaml
- .tasks/lint/
- https://raw.githubusercontent.com/taskctl/taskctl/master/docs/example.yaml
```

### Example
Config file [example](https://github.com/taskctl/taskctl/blob/master/docs/example.yaml)

### Global configuration
It is stored in ``$HOME/.taskctl/config.yaml`` file. It is handy to store global tasks, reusable contexts, defaults etc 

## Tasks
Task is a foundation of *taskctl*. It describes one or more commands to run, their environment, executors and attributes such as working directory, execution timeout, acceptance of failure, etc.
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
        after: rm -rf tmp/*
```
Task definition takes following parameters:
- ``name`` - task name (optional)
- ``command`` - one or more commands to run
- ``variations`` - list of variations to apply to command
- ``context`` - name of the context to run commands in (optional). ``local`` by default
- ``env`` - environment variables (optional). All existing environment variables will be passed automatically
- ``dir`` - working directory. If not set, current working directory will be used
- ``timeout`` - command execution timeout (optional)
- ``allow_failure`` - if set to ``true`` failed commands will no interrupt task execution. ``false`` by default
- ``after`` - command that will be executed after command completes
- ``exportAs`` - output variable name. ``TASK_NAME_OUTPUT`` by default

### Tasks variables
Each task has variables to be used to render task's fields  - `command`, `dir`.
Along with predefined, variables can be set in a task's definition.

Predefined variables are:
- ``Root`` - config file directory
- ``TaskName`` - current task's name
- ``OutputTask1`` - `task1` output

Variables can be used inside task definition. For example:
```
tasks:
    task1:
        dir: "{{ .Root }}/some-dir"
        command:
          - echo "My name is {{ .TaskName }}"
          - echo "Sleep for {{ .sleep }} seconds"
          - sleep {{ .sleep }}
        variables:
          sleep: 3
```

### Storing task's output
Task output automatically stored to the variable named like this - ``OutputTaskName``, where `TaskName` is the actual task's name.
Variable's name can be changed by a task's `exportAs` parameter.
Those variables will be available to dependent stages.

### Tasks variations
Task may run in one or more variations. Variations allows to reuse task with different env variables:
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
this config will run build 3 times with different build GOOS

## Pipelines
Pipeline is a set of stages (tasks or other pipelines) to be executed in a certain order. Stages may be executed in parallel or one-by-one. Stage may override task environment. 

This configuration:
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
          depends_on: ["task A", "task B", "finish"]
          
tasks:
    start task: ...
    task A: ...
    task B: ...
    task C: ...
    task D: ...
    task E: ...
    finish: ...
    
```
will create this pipeline:
```
               |‾‾‾ task A ‾‾‾‾‾‾‾‾‾‾‾‾‾‾|
start task --- |--- task B --------------|--- task E --- finish
               |___ task C ___ task D ___|
```
Stage definition takes following parameters:
- ``name`` - stage name (optional). If not set - referenced task or pipeline name will be used.
- ``task`` - task to execute on this stage (optional)
- ``pipeline`` - pipeline to execute on this stage (optional)
- ``env`` - environment variables (optional). All existing environment variables will be passed automatically
- ``depends_on`` - name of stage on which this stage depends on (optional). This stage will be started only after referenced stage is completed.
- ``allow_failure`` - if set to ``true`` failing stage will no interrupt pipeline execution. ``false`` by default

## Output flavors
- `raw` - raw commands output
- `formatted` - strips ANSI escape sequences where possible, prefixes command output with task name
- `cockpit` - shows only pipeline progress spinners

## Filesystem watchers
Watcher watches for changes in files selected by provided patterns and triggers a task anytime an event has occurred.
```yaml
watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"]
    exclude: ["pkg/excluded.go", "pkg/excluded-dir/*"]
    events: [create, write, remove, rename, chmod]
    task: task1
```

## Contexts
Contexts allows you to set up execution environment, shell or binaries which will run your task, up/down commands etc

### Local context with zsh
```yaml
contexts:
  local:
    executable:
      bin: /bin/zsh
      args:
        - -c
    env:
      VAR_NAME: VAR_VALUE
    before: echo "I'm local context!"
    after: echo "Have a nice day!"
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

## FAQ
### How does it differ from go-task/task?
It's amazing how solving same problems lead to same solutions. taskctl and go-task have a lot of concepts in common but also have some differences. 
1. Main is pipelines. Pipelines and stages allows more precise workflow design because same tasks may have different dependencies (or no dependencies) in different scenarios.
2. Contexts allows you to set up execution environment, shell or binaries which will run your task. Now there is several available context types: local (shell or binary), remote (ssh), container (docker, docker-compose, kubernetes via kubectl)

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
### Similar projects
- [GNU Make](https://github.com/mirror/make)
- [go-task/task](https://github.com/go-task/task)
- [mage](https://github.com/magefile/mage)
- [tusk](https://github.com/rliebz/tusk)
- [just](https://github.com/casey/just)

## How to contribute?
Feel free to contribute in any way you want. Share ideas, submit issues, create pull requests. 
You can start by improving this [README.md](https://github.com/taskctl/taskctl/blob/master/README.md) or suggesting new [features](https://github.com/taskctl/taskctl/issues)
Thank you! 

## License
This project is licensed under the GNU GPLv3 - see the [LICENSE.md](LICENSE.md) file for details

## Authors
 - Yevhen Terentiev - [trntv](https://github.com/trntv)
See also the list of [contributors](https://github.com/taskctl/taskctl/contributors) who participated in this project.
