// Package iox provides small io helpers.
package iox

import "io"

// Close closes c and discards the error. It is meant for deferred cleanup
// where the close error is not actionable (e.g. read-only files, HTTP
// response bodies), and satisfies the errcheck linter at the call site.
func Close(c io.Closer) {
	_ = c.Close()
}
