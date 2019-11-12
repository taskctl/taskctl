Warning
-------
Proof of concept, heavy work is in progress ;-)

Wilson the Task runner
----------
https://en.wikipedia.org/wiki/Cast_Away#Wilson_the_volleyball

TODO:
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
 - [ ] Links (pipeline-pipeline, task-task)
 - [ ] Task timeout
 - [ ] ssh context
 - [ ] running context
 - [ ] add "--set" flag to set config entries
 - [ ] better concurrent tasks outputs handling
 - [ ] get rid of logrus in pkg/
 - [ ] brew tap

# Build
```
go build -o wilson .
```

# Run pipeline
```
Usage:
   run [pipeline] [flags]
   run [command]

Available Commands:
  task        Run task

Flags:
  -h, --help   help for run

Global Flags:
  -c, --config string   config file to use (default "wilson.yaml")
  -d, --debug           debug

Use " run [command] --help" for more information about a command.
```

# Run task
```
Usage:
   run task [task] [flags]

Flags:
  -h, --help   help for task

Global Flags:
  -c, --config string   config file to use (default "wilson.yaml")
  -d, --debug           debug

```
