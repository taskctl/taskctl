package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/config"
	"github.com/taskctl/taskctl/internal/output"
	"github.com/taskctl/taskctl/internal/tui"
)

func newValidateCommand(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:     "validate CONFIG_FILE",
		Short:   "validates config file",
		GroupID: groupInspect,
		Args:    exactArgs(1, "validate requires exactly one config file path"),
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			loader := config.NewConfigLoader(config.NewConfig())
			_, err := loader.Load(file)

			if cfg.Output == output.FormatJSON {
				return encodeValidateJSON(file, err)
			}

			if err != nil {
				tui.Println(os.Stdout, tui.StyleError.Render("✗")+" "+file+" is invalid")
				for line := range strings.SplitSeq(strings.TrimSpace(err.Error()), "\n") {
					tui.Println(os.Stdout, tui.StyleFaint.Render("    "+line))
				}
				return reportedError{err}
			}

			tui.Println(os.Stdout, tui.StyleSuccess.Render("✓")+" "+file+" is valid")
			return nil
		},
	}
}

func encodeValidateJSON(file string, loadErr error) error {
	doc := struct {
		SchemaVersion int    `json:"schema_version"`
		Valid         bool   `json:"valid"`
		File          string `json:"file"`
		Error         string `json:"error,omitempty"`
	}{SchemaVersion: 1, Valid: loadErr == nil, File: file}
	if loadErr != nil {
		doc.Error = strings.TrimSpace(loadErr.Error())
	}

	if err := json.NewEncoder(os.Stdout).Encode(doc); err != nil {
		return err
	}
	if loadErr != nil {
		return reportedError{loadErr}
	}
	return nil
}
