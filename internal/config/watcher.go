package config

import (
	"fmt"

	"github.com/taskctl/taskctl/internal/watch"
)

func buildWatcher(name string, def *watcherDefinition, cfg *Config) (*watch.Watcher, error) {
	t, ok := cfg.Tasks[def.Task]
	if !ok {
		return nil, fmt.Errorf("watcher build failed. task %s not found", def.Task)
	}

	return watch.NewWatcher(name, def.Events, def.Watch, def.Exclude, t)
}
