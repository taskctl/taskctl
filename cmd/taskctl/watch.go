package main

import (
	"fmt"
	"sync"

	"github.com/taskctl/taskctl/pkg/output"
	"github.com/taskctl/taskctl/pkg/runner"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/pkg/watch"
)

func NewWatchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watch [WATCHERS...]",
		Short: "Start watching for filesystem events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			variables := runner.NewVariables(cfg.Variables)
			rn, err := runner.NewTaskRunner(contexts, output.FlavorFormatted, variables)
			if err != nil {
				return err
			}

			if dryRun {
				rn.DryRun()
			}

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

					err = w.Run(rn)
					if err != nil {
						logrus.Error(err)
					}
				}(w)
			}

			wg.Wait()

			return nil
		},
	}
}
