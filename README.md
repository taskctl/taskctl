Warning
-------
Proof of concept, heavy work is in progress ;-)

Wilson the Task runner
----------
Willows allows you to get rid of a bunch of bash scripts and to design you workflow pipelines in nice and neat with way 
in yaml files

[asciinema]

Install
---
``
go get -i github.com/trntv/wilson
``

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
 - [ ] testing
 - [ ] graceful shutdown
 - [ ] docker context
 - [ ] kubectl context
 - [ ] visualize pipeline (ASCII)
 - [ ] Links (pipeline-pipeline, task-task)
 - [ ] Task timeout
 - [ ] raw/silence output in task definition
 - [ ] ssh context
 - [ ] running context
 - [ ] add "--set" flag to set config entries
 - [ ] better concurrent tasks outputs handling (decorating?)
 - [ ] get rid of logrus in pkg/
 - [ ] brew tap
 - [ ] write log file on error
 - [ ] ui dashboard

# Run pipeline
```
Usage:
   run [pipeline] [flags]
   run [command]

Available Commands:
  task        Schedule task

Flags:
  -h, --help         help for run
  -q, --quiet        silence output
      --raw-output   raw output

Global Flags:
  -c, --config string   config file to use (default "wilson.yaml")
  -d, --debug           enable debug
```

# Run task
```
Usage:
   run task [task] [flags]

Flags:
  -h, --help   help for task

Global Flags:
  -c, --config string   config file to use (default "wilson.yaml")
  -d, --debug           enable debug
  -q, --quiet           silence output
      --raw-output      raw output
```


---
*waiting for inspiration
