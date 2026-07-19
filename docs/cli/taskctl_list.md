## taskctl list

lists contexts, pipelines, tasks and watchers

### Synopsis

Lists everything declared in the config. With --output json, emits a schema-versioned discovery document intended for machine/agent consumption.

```
taskctl list [flags]
```

### Examples

```
  taskctl list
  taskctl list --output json
```

### Options

```
  -h, --help   help for list
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
* [taskctl list pipelines](taskctl_list_pipelines.md)	 - list pipelines
* [taskctl list tasks](taskctl_list_tasks.md)	 - list tasks
* [taskctl list watchers](taskctl_list_watchers.md)	 - list watchers

