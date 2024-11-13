package cmd

import (
	"sync"

	"github.com/Ensono/taskctl/internal/watch"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newWatchCmd(rootCmd *TaskCtlCmd) {
	rc := &cobra.Command{
		Use:   "watch",
		Short: `watch [WATCHERS...]`,
		Long:  "starts watching for filesystem events",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			taskRunner, _, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}

			var wg sync.WaitGroup
			for _, w := range conf.Watchers {
				wg.Add(1)

				go func(w *watch.Watcher) {
					<-cancel //rootCmd.Cmd.Context().Done()
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
	}
	rootCmd.Cmd.AddCommand(rc)
}
