package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/taskctl/taskctl/internal/fsutil"
)

// skillTemplate holds the canonical SKILL.md content. It is embedded and
// injected by package main via SetSkillTemplate, because go:embed cannot reach
// the single source of truth at .agents/skills/taskctl/SKILL.md from here.
var skillTemplate string

// SetSkillTemplate injects the embedded SKILL.md content that `skill install`
// writes out. Call it once at startup before running the CLI.
func SetSkillTemplate(s string) {
	skillTemplate = s
}

func newSkillCommand() *cobra.Command {
	skillCmd := &cobra.Command{
		Use:     "skill",
		Short:   "manage AI agent skills",
		Long:    "Manages the installable taskctl AI agent skill, which teaches coding agents to drive taskctl through its machine-readable JSON interface. See `skill install`.",
		GroupID: groupSetup,
	}

	var global, force bool
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "installs the taskctl Claude Code skill",
		Long:  "Writes the taskctl SKILL.md into .claude/skills/taskctl in the current directory, or the user's home directory with --global.",
		Example: "  taskctl skill install\n" +
			"  taskctl skill install --global",
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			baseDir, err := os.Getwd()
			if global {
				baseDir, err = os.UserHomeDir()
			}
			if err != nil {
				return err
			}

			path, err := installSkill(baseDir, force)
			if err != nil {
				return err
			}

			fmt.Printf("installed: %s\n", path)
			return nil
		},
	}
	installCmd.Flags().BoolVar(&global, "global", false, "install into the user's home directory instead of the current directory")
	installCmd.Flags().BoolVar(&force, "force", false, "overwrite an existing installation")

	skillCmd.AddCommand(installCmd)

	return skillCmd
}

// installSkill writes the embedded SKILL.md to <baseDir>/.claude/skills/taskctl/SKILL.md.
// It returns the written path, or an error if the file already exists and force is false.
func installSkill(baseDir string, force bool) (string, error) {
	if skillTemplate == "" {
		return "", errors.New("skill template is empty; the binary was built without an embedded SKILL.md")
	}

	dir := filepath.Join(baseDir, ".claude", "skills", "taskctl")
	path := filepath.Join(dir, "SKILL.md")

	if !force && fsutil.FileExists(path) {
		return "", fmt.Errorf("skill already installed at %s (use --force to overwrite)", path)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(path, []byte(skillTemplate), 0o644); err != nil {
		return "", err
	}

	return path, nil
}
