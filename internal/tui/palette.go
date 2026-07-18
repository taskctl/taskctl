// Package tui holds taskctl's shared terminal-UI primitives: the color palette,
// TTY detection, styled-output helpers, and the interactive prompts (built on
// huh). Keeping them here lets cmd render without importing any Charm library
// directly. The dashboard is deliberately not here — it is a
// single-consumer bubbletea program that lives with its consumer in
// internal/output and borrows only this palette.
package tui

import "charm.land/lipgloss/v2"

// Shared style palette. These are the only colors taskctl renders with; both the
// CLI (cmd) and the output decorators reference them so the look stays uniform.
var (
	StyleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	StyleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	StyleFaint   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray
	StyleBold    = lipgloss.NewStyle().Bold(true)
	StylePrefix  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan, task-name prefix
	StyleSpinner = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow, dashboard spinner
)
