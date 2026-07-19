## taskctl show

shows a task's or pipeline's details

### Synopsis

Shows the resolved commands (for a task) or stage dependency graph (for a pipeline). With --output json, emits a schema-versioned document.

```
taskctl show TASK_OR_PIPELINE [flags]
```

### Examples

```
  taskctl show build
  taskctl show build --output json
```

### Options

```
  -h, --help   help for show
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

