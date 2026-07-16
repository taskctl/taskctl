package tui

import (
	"errors"
	"io"

	"charm.land/huh/v2"
)

// ErrAborted is returned by Select and Confirm when the user cancels the prompt
// (Ctrl-C / Esc). Callers check it with errors.Is to treat cancellation as a
// no-op rather than an error, without importing huh themselves.
var ErrAborted = errors.New("prompt aborted")

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

	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[T]().
			Title(title).
			Options(opts...).
			Value(&value),
	)).WithInput(stdin).WithAccessible(!Interactive(stdin)).Run()
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

	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(title).
			Value(&value),
	)).WithInput(stdin).WithAccessible(!Interactive(stdin)).Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, ErrAborted
		}
		return false, err
	}

	return value, nil
}
