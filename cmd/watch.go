package cmd

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/watch"
)

func newWatchCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:               "watch WATCHER [WATCHER...]",
		Short:             "starts watching for filesystem events",
		GroupID:           groupRun,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: watcherCompletion(cfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve every watcher before starting any, so an unknown name
			// fails cleanly instead of leaving earlier watchers running.
			watchers := make([]*watch.Watcher, 0, len(args))
			for _, name := range args {
				w, ok := cfg.Watchers[name]
				if !ok {
					return fmt.Errorf("unknown watcher %q", name)
				}
				watchers = append(watchers, w)
			}

			taskRunner, err := buildTaskRunner(cmd, cfg)
			if err != nil {
				return err
			}
			taskRunner.OutputFormat = output.FormatRaw

			ctx := cmd.Context()
			var wg sync.WaitGroup
			for _, w := range watchers {
				wg.Add(1)
				go func(w *watch.Watcher) {
					<-ctx.Done()
					w.Close()
				}(w)

				go func(w *watch.Watcher) {
					defer wg.Done()
					if err := w.Run(taskRunner); err != nil {
						slog.Error(err.Error())
					}
				}(w)
			}

			wg.Wait()
			return nil
		},
	}
}

func watcherCompletion(cfg *config.Config) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return completionFunc(cfg, func() []string {
		names := make([]string, 0, len(cfg.Watchers))
		for name := range cfg.Watchers {
			names = append(names, name)
		}
		return names
	})
}
