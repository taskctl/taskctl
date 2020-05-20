package main

import (
	"fmt"
	"sync"

	"github.com/taskctl/taskctl/pkg/output"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/watch"

	"github.com/sirupsen/logrus"
)

func newWatchCommand() *cli.Command {
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
			taskRunner, err := buildTaskRunner(c)
			if err != nil {
				return err
			}

			taskRunner.OutputFormat = output.FormatRaw

			var wg sync.WaitGroup
			for _, name := range c.Args().Slice() {
				wg.Add(1)
				w, ok := cfg.Watchers[name]
				if !ok {
					return fmt.Errorf("unknown watcher %s", name)
				}
				go func(w *watch.Watcher) {
					<-cancel
					w.Close()
				}(w)

				go func(w *watch.Watcher) {
					defer wg.Done()

					err = w.Run(taskRunner)
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
