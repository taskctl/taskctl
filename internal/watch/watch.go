package watch

import (
	log "github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/pkg/builder"
	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/task"
	"github.com/taskctl/taskctl/pkg/util"
	"sync"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"
)

const (
	EventCreate = "create"
	EventWrite  = "write"
	EventRemove = "remove"
	EventRename = "rename"
	EventChmod  = "chmod"
)

var fsnotifyMap = map[fsnotify.Op]string{
	fsnotify.Create: EventCreate,
	fsnotify.Write:  EventWrite,
	fsnotify.Remove: EventRemove,
	fsnotify.Rename: EventRename,
	fsnotify.Chmod:  EventChmod,
}

type Watcher struct {
	name     string
	r        *runner.TaskRunner
	finished chan struct{}
	paths    []string
	events   map[string]bool
	task     *task.Task
	fsw      *fsnotify.Watcher

	wg sync.WaitGroup
}

func BuildWatcher(name string, def *builder.WatcherDefinition, t *task.Task, r *runner.TaskRunner) (w *Watcher, err error) {
	w = &Watcher{
		name:     name,
		r:        r,
		paths:    make([]string, 0),
		finished: make(chan struct{}),
		task:     t,
		events:   make(map[string]bool),
	}

	for _, p := range def.Watch {
		matches, err := doublestar.Glob(p)
		if err != nil {
			return nil, err
		}

		for _, path := range matches {
			var excluded bool
			for _, exclude := range def.Exclude {
				matched, err := doublestar.PathMatch(exclude, path)
				if err != nil {
					return nil, err
				}

				if matched {
					excluded = true
					break
				}
			}

			if !excluded {
				w.paths = append(w.paths, path)
			}
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

	log.Debugf("starting watcher %s", w.name)
	for _, path := range w.paths {
		err = w.fsw.Add(path)
		log.Debugf("watcher %s is waiting for events in %s", w.name, path)
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
				log.Debugf("watcher %s; event %s; file: %s", w.name, event.Op.String(), event.Name)
				if event.Op == fsnotify.Rename {
					err = w.fsw.Add(event.Name)
					if err != nil {
						log.Error(err)
					}
				}
			case err, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
				log.Error(err)
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
		log.Error(err)
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

	env := util.ConvertEnv(map[string]string{
		"EVENT_NAME": eventName,
		"EVENT_PATH": event.Name,
	})

	log.Debugf("triggering %s for %s", w.task.Name, w.name)
	err := w.r.RunWithEnv(w.task, env)
	if err != nil {
		log.Error(err)
	}
}
