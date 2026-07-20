package runner

import (
	"bytes"
	"testing"

	"github.com/taskctl/taskctl/task"
	"github.com/taskctl/taskctl/variables"
)

var shBin = Binary{
	Bin:  "/bin/sh",
	Args: []string{"-c"},
}

func TestTaskCompiler_CompileCommand(t *testing.T) {
	tc := newTaskCompiler()

	job, err := tc.compileCommand(
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
	job, err = tc.compileCommand(
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

func TestTaskCompiler_CompileCommand_Typed(t *testing.T) {
	tc := newTaskCompiler()

	env := variables.FromMap(map[string]string{"FOO": "bar"})
	dockerContext := NewExecutionContext(nil, "", env, nil, nil, nil, nil, WithDocker(DockerSpec{Image: "alpine"}))

	job, err := tc.compileCommand(
		"echo hi",
		dockerContext,
		"", nil,
		&bytes.Buffer{},
		&bytes.Buffer{},
		&bytes.Buffer{},
		env,
		variables.NewVariables(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if job.Command != "docker run --rm -e 'FOO=bar' alpine sh -c 'echo hi'" {
		t.Errorf("unexpected wrapped command: %s", job.Command)
	}

	if len(job.Env.Map()) != 0 {
		t.Error("env should not be applied to the local launcher for typed contexts")
	}

	if job.Dir != "" {
		t.Error("dir should not be applied to the local launcher for typed contexts")
	}
}

func TestTaskCompiler_CompileCommand_EscapeHatchUnchanged(t *testing.T) {
	tc := newTaskCompiler()

	env := variables.FromMap(map[string]string{"HOME": "/root"})
	escapeHatchContext := NewExecutionContext(&shBin, "/tmp", env, nil, nil, nil, nil, WithQuote("'"))

	job, err := tc.compileCommand(
		"echo hi",
		escapeHatchContext,
		"/root", nil,
		&bytes.Buffer{},
		&bytes.Buffer{},
		&bytes.Buffer{},
		env,
		variables.NewVariables(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if job.Command != "/bin/sh -c 'echo hi'" {
		t.Errorf("unexpected command: %s", job.Command)
	}

	if job.Env.Map()["HOME"] != "/root" {
		t.Error("env should be applied to the local launcher for escape-hatch contexts")
	}

	if job.Dir != "/root" {
		t.Error("dir should be applied to the local launcher for escape-hatch contexts")
	}
}

func TestTaskCompiler_CompileTask(t *testing.T) {
	tc := newTaskCompiler()
	j, err := tc.compileTask(&task.Task{
		Commands:  []string{"echo 1"},
		Variables: variables.FromMap(map[string]string{"TestInterpolatedVar": "TestVar={{.TestVar}}"}),
	},
		NewExecutionContext(&shBin, "/tmp", variables.FromMap(map[string]string{"HOME": "/root"}), nil, nil, nil, nil),
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
