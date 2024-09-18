package output

import (
	"io"
	"sync"
)

// Taken from stdlib
// package io multiwriter.go
//
// NOTE:
//
// removed short write check as we want differently
//
// formatted writers
//
//	if n != len(p) {
//		err = ErrShortWrite
//		return
//	}
type multiWriter struct {
	writers []io.Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return len(p), nil
}

// MultiWriter creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
//
// Each write is written to each listed writer, one at a time.
// If a listed writer returns an error, that overall write operation
// stops and returns the error; it does not continue down the list.
func MultiWriter(writers ...io.Writer) io.Writer {
	allWriters := make([]io.Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &multiWriter{allWriters}
}

// SafeWriter is a go routine safe implementation of any io.Writer.
//
// Outside of os.File no other writer is concurrency safe,
//
// bytes.Buffer, and so on will need a concurrency lock on writes, and reads
type SafeWriter struct {
	writerImpl   io.Writer
	bytesWritten []byte
	mu           sync.Mutex
}

// NewSafeWriter initiates a new concurrency safe writer
func NewSafeWriter(writerImpl io.Writer) *SafeWriter {
	return &SafeWriter{writerImpl: writerImpl, bytesWritten: []byte{}, mu: sync.Mutex{}}
}

func (tw *SafeWriter) Write(p []byte) (n int, err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.bytesWritten = append(tw.bytesWritten, p...)
	return tw.writerImpl.Write(p)
}

// String returns the stringified version of bytes written
// Currently only used in tests
func (tw *SafeWriter) String() string {
	return string(tw.bytesWritten)
}

// Len returns the currently written bytes length
// Currently only used in tests
func (tw *SafeWriter) Len() int {
	return len(tw.bytesWritten)
}
