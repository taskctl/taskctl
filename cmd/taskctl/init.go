package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/urfave/cli/v2"

	"github.com/taskctl/taskctl/internal/config"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"

	"github.com/taskctl/taskctl/internal/util"
)

var configTmpl = `# This is an example of taskctl tasks configuration file. Adjust it to fit your needs
pipelines:
  pipeline1:
    - task: task1
    - task: task2
      depends_on: task1

tasks:
  task1:
    command: echo "I'm task1"
  
  task2:
    command: echo "I'm task2. Your date is $(date)"

watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"]
    events: [create, write, remove, rename, chmod]
    task: task1
`

func NewInitCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "init",
		Usage: "creates sample config file",
		Action: func(c *cli.Context) error {
			fileSelect := promptui.Select{
				Label: "Choose file name",
				Items: config.DefaultFileNames,
			}

			_, filename, err := fileSelect.Run()

			if err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			file := filepath.Join(cwd, filename)

			if util.FileExists(file) {
				replaceConfirmation := promptui.Prompt{
					Label:     "File already exists. Overwrite",
					IsConfirm: true,
				}

				_, err = replaceConfirmation.Run()
				if err != nil {
					if !errors.Is(err, promptui.ErrAbort) {
						return err
					}
					return nil
				}
			}

			fw, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}

			t := template.Must(template.New("init_config").Parse(configTmpl))

			err = t.Execute(fw, nil)
			if err != nil {
				return err
			}

			fmt.Println(aurora.Sprintf(aurora.Magenta("%s was created. Edit it accordingly to your needs"), aurora.Green(filename)))
			fmt.Println(aurora.Cyan("To run example pipeline - taskctl run pipeline1"))

			return nil
		},
	}

	return cmd
}
