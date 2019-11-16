# Wilson - routines automation toolkit
Willows allows you to get rid of a bunch of bash scripts and to design you workflow pipelines in nice and neat with way 
with yaml files. Wilson's automation based on four concepts:
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

## Full config example
```yaml
contexts:
    local: # will be created automatically if not set
        type: local
        executable:
          bin: /bin/bash
          args: 
            - -c
        env:
          VAR_NAME: VAR_VALUE

    docker-context-name:
        type: container
        container:
          provider: docker
          image: alpine:latest
          options:
            - -v /folder:/folder
            ...
          exec: false
          env:
            VAR_NAME: VAR_VALUE
        env:
          VAR_NAME: VAR_VALUE # eg. "DOCKER_HOST"
    
    docker-compose-context-name:
        type: container
        container:
          provider: docker-compose
          name: api
          exec: true
          options:
            - --user=root
        env:
          VAR_NAME: VAR_VALUE # eg."COMPOSE_FILE"

pipelines:
  pipeline1:
    - task: task1
    - task: task2
      depends_on: task1
    - task: task3 
      depends_on: task1 // task2 and task3 will run in parallel when task1 finished
    - task: task4
      depends_on: [task1, task2]

tasks:
    task1:
      context: local # optional. "local" is context by default
      command:
        - echo ${ARGS} # ARGS is populated by arguments passed to task. eg. wilson run task task1 -- arg1 arg2
        - echo "My name is task1"
      env:
        VAR_NAME: VAR_VALUE
      dir: /task/working/dir # current directory by default

    task2:
      context: docker-context-name
      command:
        - echo "Hello from container"
      env:
        VAR_NAME: VAR_VALUE

    task3:
      context: docker-compose-context-name # local is default context
      command:
        - echo "Hello from container created by docker-compose"
      env:
        VAR_NAME: VAR_VALUE

    task-to-be-triggered-by-watcher:
      command:
        - echo ${EVENT_NAME} ${EVENT_PATH}

watchers:
    watcher1:
      watch: ["README.*", "pkg/**/*.go"]
      events: [create, write, remove, rename, chmod]
      
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
```yaml
pipelines:
    pipeline1:
        tasks:
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

## TODO
 - [x] logrus, zap? plain formatting for log entries
 - [x] skip error when root config file not found
 - [x] pass task env to container context
 - [x] move scheduler to separate package
 - [x] global config
 
 - [x] pipelines
 - [x] env
 - [x] command env processing
 - [x] import file
 - [ ] kubectl context
 - [ ] ssh context
 - [ ] autocomplete
 - [ ] import path
 - [ ] import url
 - [ ] check for cycles in pipelines
 - [ ] tests
 - [ ] graceful shutdown +context specific
 - [x] docker context
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
 - [ ] task's env in pipeline definition

## Why "Wilson"?
https://en.wikipedia.org/wiki/Cast_Away#Wilson_the_volleyball

---
*waiting for inspiration
