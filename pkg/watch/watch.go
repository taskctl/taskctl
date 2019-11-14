package watch

import (
	"github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/config"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/task"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

const (
	EVENT_CREATE = "create"
	EVENT_WRITE  = "write"
	EVENT_REMOVE = "remove"
	EVENT_RENAME = "rename"
	EVENT_CHMOD  = "chmod"
)

var fsnotifyMap = map[fsnotify.Op]string{
	fsnotify.Create: EVENT_CREATE,
	fsnotify.Write:  EVENT_WRITE,
	fsnotify.Remove: EVENT_REMOVE,
	fsnotify.Rename: EVENT_RENAME,
	fsnotify.Chmod:  EVENT_CHMOD,
}

type Watcher struct {
	r        *runner.TaskRunner
	finished chan struct{}
	paths    []string
	events   map[string]bool
	task     *task.Task
	fsw      *fsnotify.Watcher

	wg sync.WaitGroup
}

func BuildWatcher(def *config.WatcherConfig, t *task.Task, r *runner.TaskRunner) (w *Watcher, err error) {
	w = &Watcher{
		r:        r,
		paths:    make([]string, 0),
		finished: make(chan struct{}),
		task:     t,
		events:   make(map[string]bool),
	}

	for _, p := range def.Watch {
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}

		for _, path := range matches {
			w.paths = append(w.paths, path)
		}
	}

	for _, e := range def.Events {
		w.events[e] = true
	}

	return w, nil
}

func (w *Watcher) Run() (err error) {
	w.fsw, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, path := range w.paths {
		err = w.fsw.Add(path)
		if err != nil {
			return err
		}
	}

	go func() {
		defer close(w.finished)
		for {
			select {
			case event, ok := <-w.fsw.Events:
				if !ok {
					return
				}
				w.wg.Add(1)
				go w.handle(event)
				logrus.Debugf("watch event %s", event.Name)
				if event.Op == fsnotify.Rename {
					err = w.fsw.Add(event.Name)
					if err != nil {
						logrus.Error(err)
					}
				}
			case err, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
				logrus.Error(err)
			}
		}
	}()
	w.wg.Wait()
	<-w.finished

	return nil
}

func (w *Watcher) Close() {
	err := w.fsw.Close()
	if err != nil {
		logrus.Error(err)
		return
	}
	<-w.finished
}

func (w *Watcher) handle(event fsnotify.Event) {
	defer w.wg.Done()

	eventName := fsnotifyMap[event.Op]
	if !w.events[eventName] {
		return
	}

	env := config.ConvertEnv(map[string]string{
		"EVENT_NAME": eventName,
		"EVENT_PATH": event.Name,
	})

	err := w.r.RunWithEnv(w.task, env)
	if err != nil {
		logrus.Error(err)
	}
}
