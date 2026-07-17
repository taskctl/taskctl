package main

import (
	_ "embed"
	"log/slog"
	"os"

	"github.com/taskctl/taskctl/cmd"
)

var version = "dev"

// skillTemplate is the canonical taskctl agent skill, embedded from the single
// source of truth at .agents/skills/taskctl/SKILL.md (also symlinked into
// .claude/skills/ for this repo's own agents). It is injected into the cmd
// package because go:embed cannot reach outside cmd/'s own directory.
//
//go:embed .agents/skills/taskctl/SKILL.md
var skillTemplate string

func main() {
	cmd.SetSkillTemplate(skillTemplate)

	err := cmd.Run(version)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
