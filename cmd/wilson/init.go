package main

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/trntv/wilson/pkg/util"
	"os"
	"path/filepath"
	"text/template"
)

var configTmpl = `pipelines:
  pipeline1:
    - task: task1
    - task: task2
      depends_on: task1
    - task: task3
      depends_on: task1
      env:
        GREETING: "Task 3 greeting"
    - task: task4
      depends_on: [task2, task3]

tasks:
  task1:
    command: echo "I'm task1"
  
  task2:
    command: echo "I'm task2. Your date is $(date)"
  
  task3:
    command: 
      - echo ${GREETING}
      - echo "I'm running in parallel with task2'"
  
  task4:
    command:
      - echo "SHELL is ${SHELL}"
      - echo "Goodbye!"

watchers:
  watcher1:
    watch: ["README.*", "pkg/**/*.go"]
    events: [create, write, remove, rename, chmod]
    task: task1
`

func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Init with sample config",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := util.Getcwd()
			if err != nil {
				return err
			}

			file := filepath.Join(cwd, "wilson.yaml")
			fw, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}

			t := template.Must(template.New("init_config").Parse(configTmpl))

			err = t.Execute(fw, nil)
			if err != nil {
				return err
			}

			fmt.Println(aurora.Green("wilson.yaml was successfully created. Run test pipeline with \"wilson run pipeline1\""))

			return nil
		},
	}

	return cmd
}
