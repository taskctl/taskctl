package executor

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Ensono/taskctl/pkg/output"
)

func TestDefaultExecutor_Execute(t *testing.T) {
	t.Parallel()
	b := &bytes.Buffer{}
	output := output.NewSafeWriter(b)
	e, err := NewDefaultExecutor(nil, output, io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	job1 := NewJobFromCommand("echo 'success'")
	to := 1 * time.Minute
	job1.Timeout = &to

	if _, err := e.Execute(context.Background(), job1); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "success") {
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
