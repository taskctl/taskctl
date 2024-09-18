package output_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Ensono/taskctl/pkg/output"
)

type errorWriter struct {
}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write failed")
}

func TestOutput_MultiWriter_errors(t *testing.T) {
	b := output.NewSafeWriter(&bytes.Buffer{})
	ew := &errorWriter{}
	mw := output.MultiWriter(b, ew)
	if _, err := mw.Write([]byte(`fail me`)); err == nil {
		t.Fatal("failed to throw on write")
	}
}
