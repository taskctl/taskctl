package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/util"
	"github.com/trntv/wilson/pkg/watch"
	"sync"
)

func NewWatchCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "watch [WATCHERS...]",
		Short:     "Start watching for filesystem events",
		ValidArgs: util.ListNames(cfg.Watchers),
		Args:      cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
			for _, name := range args {
				wg.Add(1)
				w, ok := watchers[name]
				if !ok {
					return fmt.Errorf("unknown watcher %s", name)
				}
				go func(w *watch.Watcher) {
					select {
					case <-cancel:
						w.Close()
						return
					}
				}(w)

				go func(w *watch.Watcher) {
					defer wg.Done()

					err := w.Run()
					if err != nil {
						log.Error(err)
					}
				}(w)
			}

			wg.Wait()

			return nil
		},
	}
}
