package tui

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"charm.land/huh/v2"
)

// ErrAborted is returned by Select and Confirm when the user cancels the prompt
// (Ctrl-C / Esc). Callers check it with errors.Is to treat cancellation as a
// no-op rather than an error, without importing huh themselves.
var ErrAborted = errors.New("prompt aborted")

// PromptReader prepares stdin for a sequence of prompts. A terminal is returned
// unchanged, so Select/Confirm drive huh's full TUI. Non-terminal stdin is
// wrapped in a *bufio.Reader that MUST be reused across every prompt in the
// sequence: each accessible prompt then pulls exactly one line from it (see
// promptInput), rather than huh's per-prompt scanner chunk-reading and swallowing
// input meant for a later prompt. This is what makes `... | taskctl init`
// scriptable across the filename + overwrite prompts.
func PromptReader(stdin io.Reader) io.Reader {
	if Interactive(stdin) {
		return stdin
	}
	return bufio.NewReader(stdin)
}

// promptInput returns the reader to hand a single accessible huh prompt. When
// stdin is the shared *bufio.Reader from PromptReader, it reads one line with
// ReadString (which leaves the remaining bytes buffered for the next prompt) and
// hands huh just that line. In every other case stdin is passed through.
func promptInput(stdin io.Reader, accessible bool) io.Reader {
	if accessible {
		if br, ok := stdin.(*bufio.Reader); ok {
			line, _ := br.ReadString('\n')
			return strings.NewReader(line)
		}
	}
	return stdin
}

// Item pairs a display label with the value returned when it is chosen. T is
// constrained to comparable because huh identifies the selected option by value.
type Item[T comparable] struct {
	Label string
	Value T
}

// StringItems builds Items whose label and value are the same string, for
// prompts that select among plain strings.
func StringItems(ss []string) []Item[string] {
	items := make([]Item[string], 0, len(ss))
	for _, s := range ss {
		items = append(items, Item[string]{Label: s, Value: s})
	}
	return items
}

// Select asks the user to pick one of items and returns the chosen value. It
// reads from stdin and drops to huh's accessible (line-based) mode when stdin
// isn't a terminal. A cancelled prompt returns ErrAborted; an empty item list
// returns a plain error instead of hanging (huh's Select makes Enter a no-op
// with zero options).
func Select[T comparable](stdin io.Reader, title string, items []Item[T]) (T, error) {
	var value T

	if len(items) == 0 {
		return value, errors.New("no options to select from")
	}

	opts := make([]huh.Option[T], 0, len(items))
	for _, it := range items {
		opts = append(opts, huh.NewOption(it.Label, it.Value))
	}

	accessible := !Interactive(stdin)
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[T]().
			Title(title).
			Options(opts...).
			Value(&value),
	)).WithInput(promptInput(stdin, accessible)).WithAccessible(accessible).Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return value, ErrAborted
		}
		return value, err
	}

	return value, nil
}

// Confirm asks a yes/no question and returns the answer. It reads from stdin and
// drops to accessible mode for non-terminals (where EOF resolves to false). A
// cancelled prompt returns ErrAborted.
func Confirm(stdin io.Reader, title string) (bool, error) {
	var value bool

	accessible := !Interactive(stdin)
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(title).
			Value(&value),
	)).WithInput(promptInput(stdin, accessible)).WithAccessible(accessible).Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, ErrAborted
		}
		return false, err
	}

	return value, nil
}
