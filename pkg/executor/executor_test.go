package executor

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	e, err := NewDefaultExecutor()
	if err != nil {
		t.Fatal(err)
	}

	job1 := NewJobFromCommand("echo 'success'")
	to := 1 * time.Minute
	job1.Timeout = &to

	output, err := e.Execute(context.Background(), job1)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(output, []byte("success")) {
		t.Error()
	}

	job1 = NewJobFromCommand("exit 1")

	_, err = e.Execute(context.Background(), job1)
	if err == nil {
		t.Error()
	}

	if _, ok := IsExitStatus(err); !ok {
		t.Error()
	}

	job2 := NewJobFromCommand("echo {{ .Fail }}")
	_, err = e.Execute(context.Background(), job2)
	if err == nil {
		t.Error()
	}

	job3 := NewJobFromCommand("printf '%s\\nLine-2\\n' '=========== Line 1 ==================' ")
	_, err = e.Execute(context.Background(), job3)
	if err != nil {
		t.Error()
	}
}
