package cmd

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
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

type initFlags struct {
	initDir  string
	noPrompt bool
}

type runnerInit struct {
	channelOut, channelErr io.Writer
}

func newInitCmd(rootCmd *TaskCtlCmd) {
	f := initFlags{}
	ri := &runnerInit{
		channelOut: rootCmd.ChannelOut,
		channelErr: rootCmd.ChannelErr,
	}
	initCmd := &cobra.Command{
		Use:   "init",
		Short: `initializes the directory with a sample config file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootCmd.viperConf.GetBool("no-prompt") && len(args) == 0 {
				return fmt.Errorf("file name must be supplied when running in non-interactive mode")
			}
			return ri.runInit(args, rootCmd.viperConf.GetString("dir"), rootCmd.viperConf.GetBool("no-prompt"))
		},
	}

	initCmd.PersistentFlags().StringVar(&f.initDir, "dir", "", "directory to initialize")
	_ = rootCmd.viperConf.BindPFlag("dir", initCmd.PersistentFlags().Lookup("dir"))

	initCmd.PersistentFlags().BoolVar(&f.noPrompt, "no-prompt", false, "do not prompt")
	_ = rootCmd.viperConf.BindPFlag("no-prompt", initCmd.PersistentFlags().Lookup("no-prompt"))

	rootCmd.Cmd.AddCommand(initCmd)
}

func (r *runnerInit) runInit(args []string, initDir string, noPrompt bool) error {
	var file string
	var accepted bool
	if initDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err //"unable to determine working directory")
		}
		initDir = cwd
	}

	// if no prompt and file names were supplied
	if len(args) > 0 && noPrompt {
		file = args[0]
		return r.writeConfig(filepath.Join(initDir, file))
	}

	selectedFile := huh.NewForm(
		huh.NewGroup(
			// select file name
			huh.NewSelect[string]().
				Title("Select config file name to write").
				Options(huh.NewOptions(config.DefaultFileNames...)...).
				Value(&file),
			// confirm write dir selection
			huh.NewConfirm().
				Title("Overwrite if exists").
				Value(&accepted),
		),
	).WithHeight(8).WithShowHelp(true)

	if err := selectedFile.Run(); err != nil {
		return err
	}
	if accepted {
		return r.writeConfig(filepath.Join(initDir, file))
	}
	return nil
}

// type OpenFile func(name string, flag int, perm fs.FileMode) (*os.File, error)

func (r *runnerInit) writeConfig(file string) error {
	fw, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	t := template.Must(template.New("init_config").Parse(configTmpl))

	err = t.Execute(fw, nil)
	if err != nil {
		return err
	}

	fmt.Fprintf(r.channelOut, "%s %s\n", fmt.Sprintf(cmdutils.GREEN_TERMINAL, file), fmt.Sprintf(cmdutils.MAGENTA_TERMINAL, "was created. Edit it accordingly to your needs"))
	fmt.Fprintf(r.channelOut, cmdutils.CYAN_TERMINAL, "To run example pipeline - taskctl run pipeline1")
	return nil
}
