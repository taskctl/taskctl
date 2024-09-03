package runner

import (
	"bytes"
	"testing"

	"github.com/ensono/taskctl/pkg/task"
	"github.com/ensono/taskctl/pkg/utils"
	"github.com/ensono/taskctl/pkg/variables"
)

var shBin = utils.Binary{
	Bin:  "/bin/sh",
	Args: []string{"-c"},
}

var envFile = utils.Envfile{
	Generate: false,
}

func TestTaskCompiler_CompileCommand(t *testing.T) {
	tc := NewTaskCompiler()

	job, err := tc.CompileCommand(
		"echo 1",
		NewExecutionContext(&shBin, "/tmp", variables.FromMap(map[string]string{"HOME": "/root"}), &envFile, nil, nil, nil, nil),
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

	quotedContext := NewExecutionContext(&shBin, "/", variables.NewVariables(), &envFile, []string{"false"}, []string{"false"}, []string{"false"}, []string{"false"}, WithQuote("\""))
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

func TestTaskCompiler_CompileTask(t *testing.T) {
	tc := NewTaskCompiler()
	j, err := tc.CompileTask(&task.Task{
		Commands:  []string{"echo 1"},
		Variables: variables.FromMap(map[string]string{"TestInterpolatedVar": "TestVar={{.TestVar}}"}),
	},
		NewExecutionContext(&shBin, "/tmp", variables.FromMap(map[string]string{"HOME": "/root"}), &envFile, nil, nil, nil, nil),
		&bytes.Buffer{},
		&bytes.Buffer{},
		&bytes.Buffer{},
		variables.NewVariables(),
		variables.FromMap(map[string]string{"TestVar": "TestVarValue"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if j.Vars.Get("TestInterpolatedVar").(string) != "TestVar=TestVarValue" {
		t.Error("var interpolation failed")
	}
}
