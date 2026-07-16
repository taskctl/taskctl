// Package tui holds taskctl's terminal-UI primitives: the shared color palette,
// TTY detection, interactive prompts (built on huh), and the cockpit dashboard
// (built on bubbletea). Keeping these in one place lets the cmd and output
// packages share styling and stay free of direct Charm dependencies.
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
	StyleSpinner = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow, cockpit spinner
)
