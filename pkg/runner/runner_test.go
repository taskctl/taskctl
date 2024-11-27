package runner_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Ensono/taskctl/internal/config"
	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/variables"

	taskpkg "github.com/Ensono/taskctl/pkg/task"
)

func TestTaskRunner(t *testing.T) {
	c := runner.NewExecutionContext(nil, "/", variables.NewVariables(), &utils.Envfile{}, []string{"true"}, []string{"false"}, []string{"echo 1"}, []string{"echo 2"})
	ob, eb := output.NewSafeWriter(&bytes.Buffer{}), output.NewSafeWriter(&bytes.Buffer{})

	rnr, err := runner.NewTaskRunner(
		runner.WithContexts(map[string]*runner.ExecutionContext{"local": c}),
		func(tr *runner.TaskRunner) {
			tr.Stdout, tr.Stderr = ob, eb

		})
	if err != nil {
		t.Fatal(err)
	}
	rnr.SetContexts(map[string]*runner.ExecutionContext{
		"default": runner.DefaultContext(),
		"local":   c,
	})

	rnr.SetVariables(variables.FromMap(map[string]string{"Root": "/tmp"}))
	rnr.WithVariable("Root", "/")

	task1 := taskpkg.NewTask("t1")
	task1.Context = "local"

	task1.Commands = []string{"echo 'taskctl'"}
	task1.Name = "some test task"
	task1.Dir = "{{.Root}}"
	task1.After = []string{"echo 'after task1'"}

	d := 1 * time.Minute
	task2 := taskpkg.NewTask("t2")
	task2.Timeout = &d
	task2.Variations = []map[string]string{{"GOOS": "windows"}, {"GOOS": "linux"}}

	task2.Commands = []string{"false"}
	task2.Name = "some test task"
	task2.Dir = "{{.Root}}"
	task2.Interactive = true

	task3 := taskpkg.NewTask("t3")
	task3.Condition = "exit 1"

	task4 := taskpkg.NewTask("t4")
	task4.Commands = []string{"function test_func() { echo \"BBB\"; } ", "test_func"}

	cases := []struct {
		t                *taskpkg.Task
		skipped, errored bool
		status           int16
		output           string
	}{
		{t: task1, output: "taskctl"},
		{t: task2, status: 1, errored: true},
		{t: task3, status: -1, skipped: true},
		{t: task4, output: "BBB"},
	}

	for _, testCase := range cases {
		err = rnr.Run(testCase.t)
		if err != nil && !testCase.errored && !testCase.skipped {
			t.Fatal(err)
		}

		if !testCase.skipped && testCase.t.Start().IsZero() {
			t.Error()
		}

		if !strings.Contains(ob.String(), testCase.output) {
			t.Error()
		}

		if testCase.errored && !testCase.t.Errored() {
			t.Error()
		}

		if !testCase.errored && testCase.t.Errored() {
			t.Error()
		}

		if testCase.t.ExitCode() != testCase.status {
			t.Error()
		}
	}

	rnr.Finish()
}

type tCloser struct {
	io.Reader
}

func (t *tCloser) Close() error {
	return nil
}

func Test_DockerExec_Cmd(t *testing.T) {
	t.Run("runs with default env file using v1 containers", func(t *testing.T) {
		dockerCtx := runner.NewExecutionContext(&utils.Binary{Bin: "docker", Args: []string{
			"run",
			"--rm",
			"alpine", "sh", "-c"}}, "/", variables.NewVariables(), utils.NewEnvFile(func(e *utils.Envfile) {}),
			[]string{""}, []string{""}, []string{""}, []string{""})

		rnr, err := runner.NewTaskRunner(runner.WithContexts(map[string]*runner.ExecutionContext{"default_docker": dockerCtx}))
		if err != nil {
			t.Fatal(err)
		}
		defer rnr.Finish()

		testOut, testErr := &bytes.Buffer{}, &bytes.Buffer{}
		rnr.Stdout, rnr.Stderr = testOut, testErr
		rnr.SetVariables(variables.FromMap(map[string]string{"Root": "/tmp"}))
		rnr.WithVariable("Root", "/")

		task1 := taskpkg.NewTask("default:docker")
		task1.Context = "default_docker"

		task1.Commands = []string{"echo 'taskctl'"}
		task1.Name = "some test task"
		task1.Dir = "{{.Root}}"
		task1.After = []string{"echo 'after task1'"}

		if err := rnr.Run(task1); err != nil {
			t.Fatalf("errored: %v\n\noutput: %v\n", err, testOut.String())
		}

		if len(testErr.String()) > 0 {
			t.Fatalf("got: %s, wanted nil", testErr.String())
		}
	})

	// with exclude
	t.Run("with exclude correctly processed using v2 containers", func(t *testing.T) {
		// Arrange
		executable := &utils.Binary{
			IsContainer: true,
			// this can be podman or any other OCI compliant deamon/runtime
			Bin:  "docker",
			Args: []string{},
		}
		executable.WithBaseArgs([]string{"run", "--rm", "--env-file"})
		executable.WithContainerArgs([]string{"-v", "${PWD}:/workspace/.taskctl", "-w", "/workspace/.taskctl", "alpine"})
		executable.WithShellArgs([]string{"sh", "-c"})

		tf, err := os.CreateTemp("", "exclude-*.env")
		if err != nil {
			t.Fatal(err)
		}

		// on program start up from Config - os.Environ are merged into contexts
		dockerCtx := runner.NewExecutionContext(executable, "/", variables.FromMap(map[string]string{"ADDED": "/old/foo", "NEW_STUFF": "/old/bar"}), utils.NewEnvFile(func(e *utils.Envfile) {
			e.PathValue = tf.Name()
			e.Exclude = append(config.DefaultContainerExcludes, "ADDED")
		}), []string{}, []string{}, []string{}, []string{})

		tf.Write([]byte(`FOO=bar
BAZ=wqiyh
QUX=looopar`))

		rnr, err := runner.NewTaskRunner(runner.WithContexts(map[string]*runner.ExecutionContext{"default_docker": dockerCtx}))
		if err != nil {
			t.Fatal(err)
		}
		defer rnr.Finish()

		testOut, testErr := &bytes.Buffer{}, &bytes.Buffer{}
		rnr.Stdout, rnr.Stderr = testOut, testErr

		task1 := taskpkg.NewTask("default:docker:with:env")
		task1.Context = "default_docker"

		task1.Commands = []string{"env"}
		task1.Name = "env test"
		task1.After = []string{"echo 'after env'"}

		// Act
		if err := rnr.Run(task1); err != nil {
			t.Fatalf("errored: %v\n\noutput: %v\n", err, testOut.String())
		}

		// Assert
		if len(testErr.String()) > 0 {
			t.Fatalf("got: %s, wanted nil", testErr.String())
		}
		rCloser := &tCloser{bytes.NewReader(testOut.Bytes())}
		got, _ := utils.ReadEnvFile(rCloser)
		if _, ok := got["ADDED"]; ok {
			t.Error("should have skipped adding var")
		}

		for _, v := range [][2]string{{"FOO", "bar"}, {"QUX", "looopar"},
			{"PWD", "/workspace/.taskctl"}, {"HOME", "/root"},
			{"NEW_STUFF", "/old/bar"}, {"BAZ", "wqiyh"}} {
			val, ok := got[v[0]]
			if !ok {
				t.Errorf("key %s not present", v[0])
			}
			if val != v[1] {
				t.Errorf("value %s not correct", v[1])
			}
		}
	})
	// 	// with custom envfile as well
}

func ExampleTaskRunner_Run() {
	t := taskpkg.FromCommands("t1", "go doc github.com/Ensono/taskctl/pkg/runner.Runner")
	ob := output.NewSafeWriter(&bytes.Buffer{})
	r, err := runner.NewTaskRunner(func(tr *runner.TaskRunner) {
		tr.Stdout = ob
	})
	if err != nil {
		return
	}
	err = r.Run(t)
	if err != nil {
		fmt.Println(err, t.ExitCode(), t.ErrorMessage())
	}
	fmt.Println(ob.String())
	// indentation is important with the matched output here
	// Output: package runner // import "github.com/Ensono/taskctl/pkg/runner"
	//
	// type Runner interface {
	// 	Run(t *task.Task) error
	// 	Cancel()
	//	Finish()
	// }
	//     Runner describes tasks runner interface
}

func TestTaskRunner_ResetContext_WithVariations(t *testing.T) {
	t.Parallel()
	ttests := map[string]struct {
		resetContext bool
		want         string
		variations   []map[string]string
	}{
		"noreset:context": {
			false,
			"first\nfirst\nfirst\nfirst\n",
			[]map[string]string{
				{"Var1": "first"}, {"Var1": "second"},
				{"Var1": "third"}, {"Var1": "fourth"},
			},
		},
		"withreset:context": {
			true,
			"first\nsecond\nthird\nfourth\n",
			[]map[string]string{
				{"Var1": "first"}, {"Var1": "second"},
				{"Var1": "third"}, {"Var1": "fourth"},
			},
		},
	}

	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			task := taskpkg.NewTask(name)
			task.Commands = []string{"echo $Var1"}
			task.ResetContext = tt.resetContext // this is set by default but setting here for clarity
			task.Variations = tt.variations

			ob, eb := output.NewSafeWriter(&bytes.Buffer{}), output.NewSafeWriter(&bytes.Buffer{})
			r, err := runner.NewTaskRunner(func(tr *runner.TaskRunner) {
				tr.Stdout = ob
				tr.Stderr = eb
			})

			if err != nil {
				t.Fatal(err)
			}

			if err := r.Run(task); err != nil {
				t.Fatal(err)
			}

			if len(ob.String()) < 1 {
				t.Error("nothing written")
			}
			if ob.String() != tt.want {
				t.Errorf("\ngot:\n%s\nwant:\n%s", ob.String(), tt.want)
			}
		})
	}
}

func TestRunner_Run_with_Artifacts(t *testing.T) {
	dir, _ := os.MkdirTemp(os.TempDir(), "artifacts*")
	defer os.RemoveAll(dir)

	stdoutBytes := &bytes.Buffer{}

	tr, err := runner.NewTaskRunner(func(tr *runner.TaskRunner) {
		tr.Stdout = output.NewSafeWriter(stdoutBytes)
	})

	if err != nil {
		t.Errorf("got: %v, wnated: <nil>", err)
	}

	taskWithArtifact := taskpkg.NewTask("with:artifact")
	taskWithArtifact.Before = []string{
		"echo 'in before command'",
	}
	taskWithArtifact.Commands = []string{
		"echo TEST_VAR=foo > .artifact.env",
	}

	taskWithArtifact.After = []string{
		"echo $TEST_VAR",
	}

	taskWithArtifact.Artifacts = &taskpkg.Artifact{
		Path: filepath.Join(dir, ".artifact.env"),
		Type: taskpkg.ArtifactType("dotenv"),
	}
	taskWithArtifact.Dir = dir
	if err := tr.Run(taskWithArtifact); err != nil {
		t.Fatal(err)
	}
	outb, _ := os.ReadFile(filepath.Join(dir, ".artifact.env"))
	if string(outb) != "TEST_VAR=foo\n" {
		t.Errorf("failed to write output in correct formant\n\ngot: %v\nwant: TEST_VAR=foo\n", string(outb))
	}
}

func TestRunner_RunWithEnvFile(t *testing.T) {
	t.Parallel()

	tf, _ := os.CreateTemp("", "ingest-*.env")
	defer os.Remove(tf.Name())
	tf.Write([]byte(`FOO=bar
BAZ=quzxxx`))

	tr, err := runner.NewTaskRunner()
	if err != nil {
		t.Fatal(err)
		return
	}
	task := taskpkg.NewTask("test:with:env")
	task.Env = task.Env.Merge(variables.FromMap(map[string]string{"ONE": "two"}))
	task.EnvFile = utils.NewEnvFile().WithPath(tf.Name())
	task.Commands = []string{"true"}

	err = tr.Run(task)
	if err != nil {
		t.Fatal(err)
	}
}
