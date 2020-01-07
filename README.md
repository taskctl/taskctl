# Wilson - routine tasks automation toolkit
![Tests](https://github.com/trntv/wilson/workflows/Test/badge.svg)
[![Requirements Status](https://requires.io/github/trntv/wilson/requirements.svg?branch=master)](https://requires.io/github/trntv/wilson/requirements/?branch=master)
![GitHub top language](https://img.shields.io/github/languages/top/trntv/wilson)
[![Go Report Card](https://goreportcard.com/badge/github.com/trntv/wilson)](https://goreportcard.com/report/github.com/trntv/wilson)

![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/trntv/wilson)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/trntv/wilson)
![GitHub closed issues](https://img.shields.io/github/issues-closed/trntv/wilson)
![GitHub issues](https://img.shields.io/github/issues/trntv/wilson)
![Licence](https://img.shields.io/github/license/trntv/wilson)

Wilson allows you to design you development workflow pipelines in nice and neat way in human-readable format (YAML, JSON or TOML). Each pipeline composed of tasks or other pipelines and allows them to run in parallel or one-by-one. 
Beside pipelines, each single task can be performed manually or triggered by filesystem watcher.

## Features
- Parallel tasks execution
- Highly customizable pipelines configuration
- stderr/stdout output capturing
- File watcher integrated with tasks and pipelines
- Customizable contexts for each task
- Human-readable configuration format (YAML, JSON or TOML)
- and many more...

[![asciicast](https://asciinema.org/a/283740.svg)](https://asciinema.org/a/283740)

# Getting started
## Install
### MacOS
```
brew tap trntv/wilson
brew install wilson
```
or
```
sudo curl -Lo /usr/local/bin/wilson https://github.com/trntv/wilson/releases/latest/download/wilson_darwin_amd64
sudo chmod +x /usr/local/bin/wilson
```

### Linux
```
sudo wget https://github.com/trntv/wilson/releases/latest/download/wilson_linux_amd64 -O /usr/local/bin/wilson
sudo chmod +x /usr/local/bin/wilson
```
### From sources
```
go get -u github.com/trntv/wilson/cmd/wilson
```

## Usage
### First run
```
wilson init
wilson run pipeline1
```
### Run task
```
wilson run task1
```
### Run pipeline
```
wilson run pipeline1
```
### Start filesystem watcher
```
wilson watch watcher1
```

## How it works?
Automation is based on four concepts:
1. [Tasks](https://github.com/trntv/wilson#tasks)
2. [Pipelines](https://github.com/trntv/wilson#pipelines) that describe set of stages (tasks or other pipelines) to run
3. [Watchers](https://github.com/trntv/wilson#watchers) that listen for filesystem events and trigger tasks
4. [Tasks contexts](https://github.com/trntv/wilson#contexts)

## Tasks
Task is a foundation of *wilson*. It describes one or more commands to run, their environment, executors and attributes such as working directory, execution timeout, acceptance of failure, etc.
```yaml
tasks:
    lint:
        command:
          - golint $(go list ./... | grep -v /vendor/)
          - go vet $(go list ./... | grep -v /vendor/)
          
    build:
        command: go build ./...
```
Task definition takes following parameters:
- ``name`` - task name (optional)
- ``command`` - one or more commands to run
- ``context`` - name of the context to run commands in (optional). ``local`` by default
- ``env`` - environment variables (optional). All existing environment variables will be passed automatically
- ``dir`` - working directory. If not set, current working directory will be used
- ``timeout`` - command execution timeout (optional)
- ``allow_failure`` - if set to ``true`` failed commands will no interrupt task execution. ``false`` by default

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

## Contexts
Available context types:
- local - shell
- container - docker, docker-compose, kubectl
- remote - ssh

## Watchers
Watcher watches for changes in files selected by provided patterns and triggers a task anytime an event has occurred.
```
watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"]
    exclude: ["pkg/excluded.go", "pkg/excluded-dir/*"]
    events: [create, write, remove, rename, chmod]
    task: task1
```

## Why "Wilson"?
In the "Cast Away" film, Wilson the volleyball 🏐 serves as Chuck Noland's (Tom Hanks) personified friend and only companion during the four years that Noland spends alone on a deserted island [wiki](https://en.wikipedia.org/wiki/Cast_Away#Wilson_the_volleyball)

## Examples
### Tasks config example
[task.yaml](https://github.com/trntv/wilson/blob/master/example/task.yaml)
```
wilson -c example/task.yaml run task echo-date-local
wilson -c example/task.yaml run task echo-date-docker
``` 
### Pipelines config example
[pipeline.yaml](https://github.com/trntv/wilson/blob/master/example/pipeline.yaml)
```
wilson -c example/pipeline.yaml run test-pipeline
wilson -c example/pipeline.yaml run pipeline1
```

### Contexts config example
[contexts.yaml](https://github.com/trntv/wilson/blob/master/example/contexts.yaml)

### Watchers config example
[watch.yaml](https://github.com/trntv/wilson/blob/master/example/watch.yaml)
```
wilson -c watch.yaml --debug watch test-watcher test-watcher-2
```

### Full config example
[full.yaml](https://github.com/trntv/wilson/blob/master/example/full.yaml)

## Autocomplete
### Bash
Add to  ~/.bashrc or ~/.profile
```
. <(wilson completion bash)
```

### ZSH
Add to  ~/.zshrc
```
. <(wilson completion zsh)
```
