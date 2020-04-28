package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"sync"

	"github.com/taskctl/taskctl/pkg/output"
	"github.com/taskctl/taskctl/pkg/runner"

	"github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/internal/watch"
)

func NewWatchCommand() *cli.Command {
	return &cli.Command{
		Name:      "watch",
		ArgsUsage: "watch [WATCHERS...]",
		Usage:     "starts watching for filesystem events",
		Before: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("no watcher specified")
			}

			return nil
		},
		Action: func(c *cli.Context) (err error) {
			rn, err := runner.NewTaskRunner(contexts, make([]string, 0), output.FlavorFormatted, c.Bool("dry-run"), cfg.Variables)

			var wg sync.WaitGroup
			for _, name := range c.Args().Slice() {
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
