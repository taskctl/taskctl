package cmd

import (
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
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
				return err
			}

			if err := cobra.OnlyValidArgs(cmd, args); err != nil {
				return err
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			var wg sync.WaitGroup
			for _, name := range args {
				wg.Add(1)
				w := watchers[name]
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
		},
	}
}
