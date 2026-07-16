package tui

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Interactive reports whether r is a terminal. huh's full TUI needs a real
// terminal; when it isn't one (a pipe, a file, CI) callers fall back to huh's
// accessible mode or skip prompting entirely, so nothing blocks on input that
// will never arrive.
func Interactive(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}

	return term.IsTerminal(int(f.Fd()))
}
