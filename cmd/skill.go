package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

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

func newSkillCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "skill",
		Usage: "manage AI agent skills",
		Subcommands: []*cli.Command{
			{
				Name:  "install",
				Usage: "installs the taskctl Claude Code skill",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "global",
						Usage: "install into the user's home directory instead of the current directory",
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "overwrite an existing installation",
					},
				},
				Action: func(c *cli.Context) error {
					var baseDir string
					var err error

					if c.Bool("global") {
						baseDir, err = os.UserHomeDir()
					} else {
						baseDir, err = os.Getwd()
					}
					if err != nil {
						return err
					}

					path, err := installSkill(baseDir, c.Bool("force"))
					if err != nil {
						return err
					}

					fmt.Printf("installed: %s\n", path)

					return nil
				},
			},
		},
	}

	return cmd
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
