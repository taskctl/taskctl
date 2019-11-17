package cmd

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/watch"
	"sync"
)

func NewWatchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watch [WATCHERS...]",
		Short: "Start watching for filesystem events",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("no watcher specified")
			}

			_, ok := tasks[args[0]]
			if !ok {
				return fmt.Errorf("unknown watcher. Available: \r\n\t%s", mapNames(cfg.Watchers))
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			tr := runner.NewTaskRunner(contexts, make([]string, 0), true, false)

			var wg sync.WaitGroup
			for _, wname := range args {
				def := cfg.Watchers[wname]
				task, ok := tasks[def.Task]
				if !ok {
					log.Fatal("task for watcher not found")
				}
				w, err := watch.BuildWatcher(wname, def, task, tr)
				if err != nil {
					log.Fatal(err)
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
						log.Error(err)
					}
				}(w)
			}

			wg.Wait()
		},
	}
}
