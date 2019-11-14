package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/runner"
	"github.com/trntv/wilson/pkg/watch"
)

func init() {
	rootCmd.AddCommand(watchCommand)
}

var watchCommand = &cobra.Command{
	Use:   "watch [watcher]",
	Short: "Start watching for filesystem events",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tr := runner.NewTaskRunner(contexts, true, false)
		def := cfg.Watchers[args[0]]
		task, ok := tasks[def.Task]
		if !ok {
			// todo: validation
			logrus.Fatal("task for watcher not found")
		}
		w, err := watch.BuildWatcher(def, task, tr)
		if err != nil {
			logrus.Fatal(err)
		}

		go func(w *watch.Watcher) {
			select {
			case <-done:
				return
			case <-cancel:
				w.Close()
				return
			}
		}(w)

		err = w.Run()
		if err != nil {
			logrus.Error()
		}

		close(done)
	},
}
