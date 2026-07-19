## taskctl completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(taskctl completion bash)

To load completions for every new session, execute once:

#### Linux:

	taskctl completion bash > /etc/bash_completion.d/taskctl

#### macOS:

	taskctl completion bash > $(brew --prefix)/etc/bash_completion.d/taskctl

You will need to start a new shell for this setup to take effect.


```
taskctl completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
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

* [taskctl completion](taskctl_completion.md)	 - Generate the autocompletion script for the specified shell

