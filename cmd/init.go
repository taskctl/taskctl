package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/fsutil"
	"github.com/taskctl/taskctl/internal/iox"
	"github.com/taskctl/taskctl/internal/tui"
)

var configTmpl = `# This is an example of taskctl tasks configuration file.
# More information at https://github.com/taskctl/taskctl
pipelines:
  pipeline1:
    - task: task1
    - task: task2
      depends_on: task1

tasks:
  task1:
    description: "Example task 1"
    command: echo "I'm task1"
  
  task2:
    description: "Example task 2"
    command: echo "I'm task2. Your date is $(date)"

watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"]
    events: [create, write, remove, rename, chmod]
    task: task1
`

func newInitCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "init",
		Usage: "creates sample config file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dir",
				Usage: "directory to initialize",
			},
		},
		Action: func(c *cli.Context) error {
			dir := c.String("dir")
			if dir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				dir = wd
			}

			// Two sequential prompts rather than one form with a conditional
			// group: huh's accessible (non-TTY) mode ignores WithHideFunc, so
			// gate the overwrite confirm in Go instead. Both prompts share one
			// PromptReader so piped input survives across them (see tui docs).
			in := tui.PromptReader(stdin)

			var filename string
			if nonInteractive(c) {
				filename = config.DefaultFileNames[0]
			} else {
				var err error
				filename, err = tui.Select(in, "Choose file name", tui.StringItems(config.DefaultFileNames))
				if err != nil {
					if errors.Is(err, tui.ErrAborted) {
						return nil
					}
					return err
				}
			}

			file := filepath.Join(dir, filename)
			if fsutil.FileExists(file) {
				if nonInteractive(c) {
					return fmt.Errorf("%s already exists; remove it or run interactively to overwrite", file)
				}

				overwrite, err := tui.Confirm(in, fmt.Sprintf("%s already exists. Overwrite?", filename))
				if err != nil {
					if errors.Is(err, tui.ErrAborted) {
						return nil
					}
					return err
				}

				if !overwrite {
					return nil
				}
			}

			fw, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer iox.Close(fw)

			t := template.Must(template.New("init_config").Parse(configTmpl))

			err = t.Execute(fw, nil)
			if err != nil {
				return err
			}

			tui.Println(os.Stdout, tui.StyleSuccess.Render(fmt.Sprintf("%s was created. Edit it accordingly to your needs", filename)))
			tui.Println(os.Stdout, tui.StyleFaint.Render("To Run example pipeline - taskctl Run pipeline1"))

			return nil
		},
	}

	return cmd
}
