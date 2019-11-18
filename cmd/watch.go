package cmd

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/util"
	"github.com/trntv/wilson/pkg/watch"
	"strings"
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

			for _, arg := range args {
				_, ok := watchers[arg]
				if !ok {
					return fmt.Errorf("unknown watcher. Available: %s\r\n", strings.Join(util.ListNames(cfg.Watchers), ", "))
				}
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
