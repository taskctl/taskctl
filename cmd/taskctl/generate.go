package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ensono/taskctl/internal/cmdutils"
	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/genci"
	"github.com/Ensono/taskctl/internal/utils"
	"github.com/spf13/cobra"
)

type generateFlags struct {
	targetTyp string
	outputDir string
}

func newGenerateCmd(rootCmd *TaskCtlCmd) {
	f := &generateFlags{}
	c := &cobra.Command{
		Use:          "generate",
		Aliases:      []string{"ci", "gen-ci"},
		Short:        `generate <pipeline>`,
		Example:      `taskctl generate pipeline1`,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := rootCmd.initConfig()
			if err != nil {
				return err
			}
			// display selector if nothing is supplied
			if len(args) == 0 {
				selected, err := cmdutils.DisplayTaskSelection(conf, true)
				if err != nil {
					return err
				}
				args = append([]string{selected}, args[0:]...)
			}

			_, argsStringer, err := rootCmd.buildTaskRunner(args, conf)
			if err != nil {
				return err
			}
			return generateDefinition(conf, argsStringer, f)
		},
	}

	c.Flags().StringVarP(&f.targetTyp, "target", "t", "", "Target type of the generation. Valid values include github, etc...")
	_ = c.MarkFlagRequired("target")
	c.Flags().StringVarP(&f.outputDir, "output", "", "", "Output directory where to create the generated file(s). Default value varies by target - e.g. github => .github/workflows")
	rootCmd.Cmd.AddCommand(c)
}

var DefaultCIOutput = map[genci.CITarget]string{
	genci.GitHubCITarget: ".github/workflows",
}

func generateDefinition(conf *config.Config, argsStringer *argsToStringsMapper, f *generateFlags) (err error) {
	pipeline := argsStringer.pipelineName
	if pipeline == nil {
		return fmt.Errorf("specified arg is not a pipeline")
	}

	g, err := genci.New(genci.CITarget(f.targetTyp), conf)
	if err != nil {
		return err
	}

	b, err := g.Convert(pipeline)
	if err != nil {
		return err
	}

	output := f.outputDir
	if output == "" {
		// lookup the default path - it must be of a valid target
		if defaultPath, ok := DefaultCIOutput[genci.CITarget(f.targetTyp)]; ok {
			output = defaultPath
		}
	}

	file, err := os.Create(filepath.Join(output, fmt.Sprintf("%s.yml", utils.ConvertToMachineFriendly(pipeline.Name()))))
	if err != nil {
		return err
	}
	if _, err := file.Write(b); err != nil {
		return err
	}
	return nil
}
