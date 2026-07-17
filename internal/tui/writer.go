package tui

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
)

// NewWriter wraps w in a colorprofile.Writer that downsamples ANSI color to
// whatever the destination supports (stripping it entirely for non-terminals).
// The wrapper is stateless across writes, so callers should build it once and
// reuse it rather than per line.
func NewWriter(w io.Writer) io.Writer {
	return colorprofile.NewWriter(w, os.Environ())
}
