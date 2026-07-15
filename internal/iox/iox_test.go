package iox

import (
	"errors"
	"testing"
)

type fakeCloser struct {
	closed bool
	err    error
}

func (f *fakeCloser) Close() error {
	f.closed = true
	return f.err
}

func TestClose(t *testing.T) {
	t.Run("calls Close", func(t *testing.T) {
		c := &fakeCloser{}
		Close(c)
		if !c.closed {
			t.Error("Close was not called on the underlying closer")
		}
	})

	t.Run("discards error", func(t *testing.T) {
		// Must not panic or propagate; the error is intentionally ignored.
		Close(&fakeCloser{err: errors.New("boom")})
	})
}
