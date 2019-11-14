package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/watch"
	"sync"
)

func NewWatchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watch [WATCHERS...]",
		Short: "Start watching for filesystem events",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			tr := runner.NewTaskRunner(contexts, make([]string, 0), true, false)

			var wg sync.WaitGroup
			for _, wname := range args {
				def := cfg.Watchers[wname]
				task, ok := tasks[def.Task]
				if !ok {
					// todo: validation
					logrus.Fatal("task for watcher not found")
				}
				w, err := watch.BuildWatcher(def, task, tr)
				if err != nil {
					logrus.Fatal(err)
				}

				go func(w *watch.Watcher) {
					select {
					case <-cancel:
						w.Close()
						return
					}
				}(w)

				go func(w *watch.Watcher) {
					wg.Add(1)
					defer wg.Done()

					err = w.Run()
					if err != nil {
						logrus.Error()
					}
				}(w)
			}

			wg.Wait()
		},
	}
}
