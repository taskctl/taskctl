## taskctl init

creates sample config file

### Synopsis

Writes a sample tasks.yaml with an example pipeline, task, and watcher to get started from.

```
taskctl init [flags]
```

### Examples

```
  taskctl init
  taskctl init --dir ./sub
```

### Options

```
      --dir string   directory to initialize
  -h, --help         help for init
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

