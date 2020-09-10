package runner

import (
	"bytes"
	"testing"

	"github.com/taskctl/taskctl/pkg/utils"
	"github.com/taskctl/taskctl/pkg/variables"
)

var shBin = utils.Binary{
	Bin:  "/bin/sh",
	Args: []string{"-c"},
}

func TestTaskCompiler_CompileCommand(t *testing.T) {
	tc := NewTaskCompiler()

	job, err := tc.CompileCommand(
		"echo 1",
		NewExecutionContext(&shBin, "/tmp", variables.FromMap(map[string]string{"HOME": "/root"}), nil, nil, nil, nil),
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

	quotedContext := NewExecutionContext(&shBin, "/", variables.NewVariables(), []string{"false"}, []string{"false"}, []string{"false"}, []string{"false"}, WithQuote("\""))
	job, err = tc.CompileCommand(
		"echo 1",
		quotedContext,
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

	if job.Command != "/bin/sh -c \"echo 1\"" {
		t.Error("task with context wasn't quoted")
	}
}
