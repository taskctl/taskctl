## taskctl run

run one or more pipelines or tasks

### Synopsis

Runs one or more named pipelines or tasks in order, stopping at the first failure. Arguments after "--" are passed to each task via the `.Args`/`.ArgsList` template variables or the `ARGS` environment variable.

```
taskctl run TARGET [TARGET...] [-- task-args] [flags]
```

### Examples

```
  taskctl run pipeline1
  taskctl run task1 task2
  taskctl run test -- -v
```

### Options

```
  -h, --help   help for run
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

* [taskctl](taskctl.md)	 - modern task runner
* [taskctl run task](taskctl_run_task.md)	 - run one or more tasks

