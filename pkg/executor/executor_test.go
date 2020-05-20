package executor

import (
	"bytes"
	"context"
	"testing"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	e, err := NewDefaultExecutor()
	if err != nil {
		t.Fatal(err)
	}

	job := NewJobFromCommand("echo 'success'")

	output, err := e.Execute(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(output, []byte("success")) {
		t.Error()
	}

	job = NewJobFromCommand("exit 1")

	_, err = e.Execute(context.Background(), job)
	if err == nil {
		t.Error()
	}

	if _, ok := IsExitStatus(err); !ok {
		t.Error()
	}
}
