package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/taskctl/taskctl/cmd"
)

func run(dir string) error {
	root := cmd.NewRootCommand("dev")
	root.DisableAutoGenTag = true // suppress the timestamp footer so generated docs are reproducible

	// cobra wires these only when a command actually executes; initialize them up
	// front so the generated reference covers the full shipped CLI surface.
	root.InitDefaultVersionFlag()
	root.InitDefaultCompletionCmd()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return doc.GenMarkdownTree(root, dir)
}

func main() {
	dir := "docs/cli"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	if err := run(dir); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
