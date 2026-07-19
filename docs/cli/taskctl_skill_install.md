## taskctl skill install

installs the taskctl Claude Code skill

### Synopsis

Writes the taskctl SKILL.md into .claude/skills/taskctl in the current directory, or the user's home directory with --global.

```
taskctl skill install [flags]
```

### Examples

```
  taskctl skill install
  taskctl skill install --global
```

### Options

```
      --force    overwrite an existing installation
      --global   install into the user's home directory instead of the current directory
  -h, --help     help for install
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

* [taskctl skill](taskctl_skill.md)	 - manage AI agent skills

