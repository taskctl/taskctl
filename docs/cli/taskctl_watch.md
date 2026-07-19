## taskctl watch

starts watching for filesystem events

### Synopsis

Watches the filesystem paths declared by one or more named watchers, running their task on matching create/write/remove/rename/chmod events until interrupted.

```
taskctl watch WATCHER [WATCHER...] [flags]
```

### Examples

```
  taskctl watch watcher1 watcher2
```

### Options

```
  -h, --help   help for watch
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

