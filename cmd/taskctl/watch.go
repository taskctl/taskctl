package main

import (
	"fmt"
	"sync"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/watch"

	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/runner"

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
			taskRunner, err := runner.NewTaskRunner(contexts, output.FlavorFormatted, cfg.Variables)
			if err != nil {
				return err
			}

			if c.Bool("dry-run") {
				taskRunner.DryRun()
			}

			var wg sync.WaitGroup
			for _, name := range c.Args().Slice() {
				wg.Add(1)
				w, ok := watchers[name]
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
