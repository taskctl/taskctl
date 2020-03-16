package task

import (
	"bytes"
)

type log struct {
	buf bytes.Buffer
}

func (l *log) String() string {
	return l.buf.String()
}

func (l *log) Write(p []byte) (int, error) {
	ln := len(p)
	pb := make([]byte, ln)
	copy(pb, p)
	pb = append(pb, "\r\n"...)
	_, err := l.buf.Write(pb)

	return ln, err
}

func (l *log) Read(p []byte) (n int, err error) {
	return l.buf.Read(p)
}

func (l *log) Len() int {
	return l.buf.Len()
}
