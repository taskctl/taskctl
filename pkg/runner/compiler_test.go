package runner

import (
	"bytes"
	"testing"

	"github.com/taskctl/taskctl/pkg/utils"
	"github.com/taskctl/taskctl/pkg/variables"
)

func TestTaskCompiler_CompileCommand(t *testing.T) {
	tc := NewTaskCompiler()

	job, err := tc.CompileCommand(
		"echo 1",
		NewExecutionContext(&utils.Binary{
			Bin:  "/bin/sh",
			Args: []string{"-c"},
		}, "/tmp", variables.FromMap(map[string]string{"HOME": "/root"}), nil, nil, nil, nil),
		"/root", nil,
		&bytes.Buffer{},
		&bytes.Buffer{},
		&bytes.Buffer{},
		variables.NewVariables(),
		variables.NewVariables(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if job.Dir != "/root" {
		t.Error()
	}

	if job.Command != "/bin/sh -c echo 1" {
		t.Error()
	}
}
