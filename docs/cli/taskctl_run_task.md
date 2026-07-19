## taskctl run task

run one or more tasks

### Synopsis

Runs one or more named tasks directly, rejecting pipeline names (unlike plain `run`).

```
taskctl run task TASK [TASK...] [-- task-args] [flags]
```

### Examples

```
  taskctl run task test -- -v
  taskctl run task task1 task2
```

### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
  -c, --config string   config file to use (default tasks.yaml or taskctl.yaml)
  -d, --debug           enable debug
      --dry-run         dry run
      --no-input        disable interactive prompts
  -o, --output string   output format (default, prefixed, raw or json)
  -q, --quiet           quiet mode
  -r, --raw             shortcut for --output=raw
      --set strings     set global variable value
  -s, --summary         show summary (default true)
```

### SEE ALSO

* [taskctl run](taskctl_run.md)	 - run one or more pipelines or tasks

