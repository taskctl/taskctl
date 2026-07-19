## taskctl list pipelines

list pipelines

### Synopsis

Lists pipeline names one per line; with --output json, a schema-versioned array of pipeline summaries with their stages.

```
taskctl list pipelines [flags]
```

### Examples

```
  taskctl list pipelines
```

### Options

```
  -h, --help   help for pipelines
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

