package executor

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	e, err := NewDefaultExecutor()
	if err != nil {
		t.Fatal(err)
	}

	job := NewJobFromCommand("echo 'success'")
	job.Stdout, job.Stderr = ioutil.Discard, ioutil.Discard

	output, err := e.Execute(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(output, []byte("success")) {
		t.Fatal()
	}
}
