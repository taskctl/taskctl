## taskctl

modern task runner

### Synopsis

taskctl runs pipelines and tasks declared in tasks.yaml (or taskctl.yaml). Running it with one or more target names executes them directly; with no target and a TTY it opens an interactive selector. See the global flags for output format, config file location, and non-interactive/dry-run controls.

```
taskctl [target...] [-- task-args] [flags]
```

### Examples

```
  taskctl prepare
  taskctl list --output json
  taskctl run test -- -v
```

### Options

```
  -c, --config string   config file to use (default tasks.yaml or taskctl.yaml)
  -d, --debug           enable debug
      --dry-run         dry run
  -h, --help            help for taskctl
      --no-input        disable interactive prompts
  -o, --output string   output format (default, prefixed, raw or json)
  -q, --quiet           quiet mode
  -r, --raw             shortcut for --output=raw
      --set strings     set global variable value
  -s, --summary         show summary (default true)
  -v, --version         version for taskctl
```

### SEE ALSO

* [taskctl completion](taskctl_completion.md)	 - Generate the autocompletion script for the specified shell
* [taskctl graph](taskctl_graph.md)	 - visualizes pipeline execution graph
* [taskctl init](taskctl_init.md)	 - creates sample config file
* [taskctl list](taskctl_list.md)	 - lists contexts, pipelines, tasks and watchers
* [taskctl run](taskctl_run.md)	 - run one or more pipelines or tasks
* [taskctl show](taskctl_show.md)	 - shows a task's or pipeline's details
* [taskctl skill](taskctl_skill.md)	 - manage AI agent skills
* [taskctl validate](taskctl_validate.md)	 - validates config file
* [taskctl watch](taskctl_watch.md)	 - starts watching for filesystem events

