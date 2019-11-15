# Wilson - routines automation toolkit
Willows allows you to get rid of a bunch of bash scripts and to design you workflow pipelines in nice and neat with way 
with yaml files. Wilson's automation based on four pillars:
1. Execution contexts
2. Tasks
3. Pipelines that describe set of tasks to run
4. Optional watchers that listen for filesystem events and trigger tasks

## Warning
Proof of concept, heavy work is in progress ;-)

## Install
### MacOS
```
brew tap trntv/wilson https://github.com/trntv/wilson.git
brew install trntv/wilson/wilson
```
### Linux
```
curl -L https://github.com/trntv/wilson/releases/latest/download/wilson-linux-amd64.tar.gz | tar xz
```
### From sources
```
go get -i github.com/trntv/wilson
```
## Tasks
```
tasks:
  task-name:
    context: api
    command:
      - some build command
      - some other build command
    env:
        BUILD_ENV: dev
        OTHER_VAR: 42  
```

## Contexts
Available context types:
- local - shell
- container - docker, docker-compose, kubectl
- remote - ssh

## Pipelines
This configuration:
```
pipelines:
    pipeline1:
        tasks:
            - task: start task
            - task: task A
              depends: ["start task"]
            - task: task B
              depends: ["start task"]
            - task: task C
              depends: ["start task"]
            - task: task D
              depends: ["task C"]
            - task: task E
              depends: ["task A", "task B", "task D"]
            - task: finish
              depends: ["task A", "task B", "finish"]
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

## TODO
 - [x] pipelines
 - [x] env
 - [x] command env processing
 - [x] import file
 - [ ] autocomplete
 - [ ] import path
 - [ ] import url
 - [ ] global config
 - [ ] check for cycles in pipelines
 - [ ] tests
 - [ ] graceful shutdown +context specific
 - [ ] set task env with -e in container context
 - [x] docker context
 - [ ] kubectl context
 - [ ] ssh context
 - [ ] visualize pipeline (ASCII)
 - [ ] links (pipeline-pipeline, task-task)
 - [ ] task timeout
 - [ ] raw/silence output in task definition
 - [ ] context preparation
 - [ ] add "--set" flag to replace config params
 - [x] better concurrent tasks outputs handling (decorating?)
 - [X] brew formula
 - [ ] write log file on error
 - [ ] ui dashboard
 - [ ] task and command as string
 - [ ] task's args in pipeline definition

## Why "Wilson"?
https://en.wikipedia.org/wiki/Cast_Away#Wilson_the_volleyball

---
*waiting for inspiration
