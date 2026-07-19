## taskctl validate

validates config file

### Synopsis

Loads the given config file and reports whether it parses and resolves cleanly, without running anything.

```
taskctl validate CONFIG_FILE [flags]
```

### Examples

```
  taskctl validate tasks.yaml
```

### Options

```
  -h, --help   help for validate
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

