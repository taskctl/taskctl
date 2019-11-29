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

Wilson allows you to design you development workflow pipelines in nice and neat way in yaml files. Each pipeline composed from tasks or other pipelines and allows them to run in parallel or one-by-one. Tasks may be runned manually or triggered by filesystem watcher.

Automation is based on four concepts:
1. Task
2. Pipeline that describes set of stages (tasks, other pipeline) to run
3. Optional watcher that listens for filesystem events and triggers tasks
4. Execution context

[![asciicast](https://asciinema.org/a/283740.svg)](https://asciinema.org/a/283740)

# Getting started
## Install
### MacOS
```
brew install trntv/wilson/wilson
```
### Linux
```
curl -L https://github.com/trntv/wilson/releases/latest/download/wilson-linux-amd64.tar.gz | tar xz
```
### From sources
```
go get -u github.com/trntv/wilson
```

### First run
```
wilson init
wilson run pipeline1
```

## Examples
### Tasks
[Task config](https://github.com/trntv/wilson/blob/master/example/task.yaml)
```
wilson -c example/task.yaml run task echo-date-local
wilson -c example/task.yaml run task echo-date-docker
``` 
### Pipelines
[Pipelines config](https://github.com/trntv/wilson/blob/master/example/pipeline.yaml)
```
wilson -c example/pipeline.yaml run test-pipeline
wilson -c example/pipeline.yaml run pipeline1
```

### Contexts
[Contexts config](https://github.com/trntv/wilson/blob/master/example/contexts.yaml)

### Watchers
[Watchers config](https://github.com/trntv/wilson/blob/master/example/watchers.yaml)
```
wilson -c watch.yaml --debug watch test-watcher test-watcher-2
```

### Full config
[Full config example](https://github.com/trntv/wilson/blob/master/example/full.yaml)

## Contexts
Available context types:
- local - shell
- container - docker, docker-compose, kubectl
- remote - ssh

## Pipelines
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

## Watchers
WIF*

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

## Why "Wilson"?
https://en.wikipedia.org/wiki/Cast_Away#Wilson_the_volleyball 🏐

---
*waiting for inspiration
