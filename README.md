# Wilson - routines automation toolkit
Willows allows you to get rid of a bunch of bash scripts and to design you workflow pipelines in nice and neat with way 
with yaml files. 

## Warning
Proof of concept, heavy work is in progress ;-)

##Install
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

Contexts
---
WIF*

Tasks
---
WIF*

Pipelines
---
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

Watchers
---
WIF*

TODO
---
 - [x] pipelines
 - [x] env
 - [x] command env processing
 - [x] import file
 - [ ] autocomplete
 - [ ] import path
 - [ ] import url
 - [ ] global config
 - [ ] check for cycles
 - [ ] tests
 - [ ] graceful shutdown
 - [ ] set task env with -e in container context
 - [x] docker context
 - [ ] kubectl context
 - [ ] visualize pipeline (ASCII)
 - [ ] Links (pipeline-pipeline, task-task)
 - [ ] Task timeout
 - [ ] raw/silence output in task definition
 - [ ] ssh context
 - [ ] running context
 - [ ] add "--set" flag to set config entries
 - [x] better concurrent tasks outputs handling (decorating?)
 - [X] brew formula
 - [ ] write log file on error
 - [ ] ui dashboard
 - [ ] task and command as string

---
*waiting for inspiration
