package cmd

import (
	"fmt"
	"sync"

	"github.com/Ensono/taskctl/internal/watch"
	outputPkg "github.com/Ensono/taskctl/pkg/output"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	newWatchCommand = &cobra.Command{
		Use:   "watch",
		Short: `watch [WATCHERS...]`,
		Long:  "starts watching for filesystem events",
		Args:  cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			output = outputPkg.FormatRaw
			if err := initConfig(); err != nil {
				return err
			}
			return buildTaskRunner(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var wg sync.WaitGroup
			for _, name := range args {
				wg.Add(1)
				w, ok := conf.Watchers[name]
				if !ok {
					return fmt.Errorf("unknown watcher %s", name)
				}
				go func(w *watch.Watcher) {
					<-cancel
					w.Close()
				}(w)

				go func(w *watch.Watcher) {
					defer wg.Done()

					err := w.Run(taskRunner)
					if err != nil {
						logrus.Error(err)
					}
				}(w)
			}

			wg.Wait()

			return nil
		},
		PostRunE: func(cmd *cobra.Command, args []string) error {
			return postRunReset()
		},
	}
)

func init() {
	TaskCtlCmd.AddCommand(newWatchCommand)
}

// func newWatchCommand() *cli.Command {
// 	return &cli.Command{
// 		Name:      "watch",
// 		ArgsUsage: "watch [WATCHERS...]",
// 		Usage:     "starts watching for filesystem events",
// 		Before: func(c *cli.Context) error {
// 			if c.NArg() == 0 {
// 				return fmt.Errorf("no watcher specified")
// 			}

// 			return nil
// 		},
// 		Action: func(c *cli.Context) (err error) {
// 			taskRunner, err := buildTaskRunner(c)
// 			if err != nil {
// 				return err
// 			}

// 			taskRunner.OutputFormat = output.FormatRaw

// 			var wg sync.WaitGroup
// 			for _, name := range c.Args().Slice() {
// 				wg.Add(1)
// 				w, ok := cfg.Watchers[name]
// 				if !ok {
// 					return fmt.Errorf("unknown watcher %s", name)
// 				}
// 				go func(w *watch.Watcher) {
// 					<-cancel
// 					w.Close()
// 				}(w)

// 				go func(w *watch.Watcher) {
// 					defer wg.Done()

// 					err = w.Run(taskRunner)
// 					if err != nil {
// 						logrus.Error(err)
// 					}
// 				}(w)
// 			}

// 			wg.Wait()

// 			return nil
// 		},
// 	}
// }
