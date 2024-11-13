# Watchers

## Filesystem watchers

Watcher watches for changes in files selected by provided patterns and triggers task anytime an event has occurred.
```yaml
watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"] # Files to watch
    exclude: ["pkg/excluded.go", "pkg/excluded-dir/*"] # Exclude patterns
    events: [create, write, remove, rename, chmod] # Filesystem events to listen to
    task: task1 # Task to run when event occurs
```

### Patterns

Thanks to [doublestar](https://github.com/bmatcuk/doublestar) *taskctl* supports the following special terms within include and exclude patterns:

Special Terms | Meaning
------------- | -------
`*`           | matches any sequence of non-path-separators
`**`          | matches any sequence of characters, including path separators
`?`           | matches any single non-path-separator character
`[class]`     | matches any single non-path-separator character against a class of characters ([details](https://github.com/bmatcuk/doublestar/blob/master/README.md#character-classes))
`{alt1,...}`  | matches a sequence of characters if one of the comma-separated alternatives matches

Any character with a special meaning can be escaped with a backslash (`\`).