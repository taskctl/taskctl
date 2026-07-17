package tui

import (
	"io"

	"charm.land/lipgloss/v2"
)

// Println writes s to w followed by a newline, downsampling any embedded ANSI
// color to what w supports. Render styled text with a palette style first, e.g.
// tui.Println(w, tui.StyleSuccess.Render("done")).
func Println(w io.Writer, s string) {
	_, _ = lipgloss.Fprintln(w, s)
}

// Printf writes a formatted line to w, downsampling embedded ANSI color to what
// w supports. Use with pre-rendered palette styles in the arguments.
func Printf(w io.Writer, format string, a ...any) {
	_, _ = lipgloss.Fprintf(w, format, a...)
}
