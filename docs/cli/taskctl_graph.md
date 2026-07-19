## taskctl graph

visualizes pipeline execution graph

### Synopsis

Generates a visual representation of pipeline execution plan. The output is in the DOT format, which can be used by GraphViz to generate charts.

```
taskctl graph PIPELINE [flags]
```

### Examples

```
  taskctl graph pipeline1 | dot -Tsvg > graph.svg
```

### Options

```
  -h, --help   help for graph
      --lr     orients the output graph left-to-right
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

