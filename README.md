# Wilson - routines automation toolkit
Willows allows you to get rid of a bunch of bash scripts and to design you workflow pipelines in nice and neat with way 
with yaml files. Wilson's automation based on four concepts:
1. Execution context
2. Task
3. Pipeline that describes set of tasks to run
4. Optional watcher that listens for filesystem events and trigger tasks

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
go get -u github.com/trntv/wilson
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
[Watchers config](https://github.com/trntv/wilson/blob/master/example/contexts.yaml)
```
wilson -c watch.yaml --debug watch test-watcher test-watcher-2
```

### Full config
[Full config example](https://github.com/trntv/wilson/blob/master/example/contexts.yaml)

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

## Why "Wilson"?
https://en.wikipedia.org/wiki/Cast_Away#Wilson_the_volleyball

---
*waiting for inspiration
