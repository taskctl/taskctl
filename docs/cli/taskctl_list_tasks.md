## taskctl list tasks

list tasks

### Synopsis

Lists task names one per line; with --output json, a schema-versioned array of task summaries.

```
taskctl list tasks [flags]
```

### Examples

```
  taskctl list tasks
```

### Options

```
  -h, --help   help for tasks
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

* [taskctl list](taskctl_list.md)	 - lists contexts, pipelines, tasks and watchers

