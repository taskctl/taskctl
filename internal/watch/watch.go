package watch

import (
	"github.com/taskctl/taskctl/pkg/variables"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/bmatcuk/doublestar"
	"github.com/fsnotify/fsnotify"

	"github.com/taskctl/taskctl/pkg/runner"
	"github.com/taskctl/taskctl/pkg/task"
)

const (
	eventCreate = "create"
	eventWrite  = "write"
	eventRemove = "remove"
	eventRename = "rename"
	eventChmod  = "chmod"
)

var fsnotifyMap = map[fsnotify.Op]string{
	fsnotify.Create: eventCreate,
	fsnotify.Write:  eventWrite,
	fsnotify.Remove: eventRemove,
	fsnotify.Rename: eventRename,
	fsnotify.Chmod:  eventChmod,
}

// Watcher is a file watcher. It triggers tasks or pipelines when filesystem event occurs
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

// NewWatcher creates new Watcher instance
func NewWatcher(name string, events, watch, exclude []string, t *task.Task) (w *Watcher, err error) {
	w = &Watcher{
		name:     name,
		paths:    make([]string, 0),
		finished: make(chan struct{}),
		task:     t,
		events:   make(map[string]bool),
	}

	for _, p := range watch {
		matches, err := doublestar.Glob(p)
		if err != nil {
			return nil, err
		}

		for _, path := range matches {
			var excluded bool
			for _, exclude := range exclude {
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

	if len(events) == 0 {
		events = []string{eventCreate, eventWrite, eventRemove, eventRename, eventChmod}
	}

	for _, e := range events {
		w.events[e] = true
	}

	return w, nil
}

// Run starts file watcher with provided TaskRunner
func (w *Watcher) Run(r *runner.TaskRunner) (err error) {
	w.r = r
	w.fsw, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	logrus.Debugf("starting watcher %s", w.name)
	for _, path := range w.paths {
		err = w.fsw.Add(path)
		logrus.Debugf("watcher %s is waiting for events in %s", w.name, path)
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
				logrus.Debugf("watcher %s; event %s; file: %s", w.name, event.Op.String(), event.Name)
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

// Close  stops this watcher
func (w *Watcher) Close() {
	if w.fsw != nil {
		err := w.fsw.Close()
		if err != nil {
			logrus.Error(err)
		}
	}
	<-w.finished
}

func (w *Watcher) handle(event fsnotify.Event) {
	defer w.wg.Done()

	eventName := fsnotifyMap[event.Op]
	if !w.events[eventName] {
		return
	}

	logrus.Debugf("triggering %s for %s", w.task.Name, w.name)

	t := *w.task
	t.Env = t.Env.Merge(variables.FromMap(map[string]string{
		"EventName": eventName,
		"EventPath": event.Name,
	}))

	t.Variables = t.Variables.Merge(variables.FromMap(map[string]string{
		"EVENT_NAME": eventName,
		"EVENT_PATH": event.Name,
	}))

	err := w.r.Run(&t)
	if err != nil {
		logrus.Error(err)
	}
}
