package runner_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Ensono/taskctl/internal/utils"
	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/Ensono/taskctl/pkg/variables"
)

func TestContext(t *testing.T) {
	logrus.SetOutput(io.Discard)

	c1 := runner.NewExecutionContext(nil, "/", variables.NewVariables(), &utils.Envfile{}, []string{"true"}, []string{"false"}, []string{"true"}, []string{"false"})
	c2 := runner.NewExecutionContext(nil, "/", variables.NewVariables(), &utils.Envfile{}, []string{"false"}, []string{"false"}, []string{"false"}, []string{"false"})

	runner, err := runner.NewTaskRunner(runner.WithContexts(map[string]*runner.ExecutionContext{"after_failed": c1, "before_failed": c2}))
	if err != nil {
		t.Fatal(err)
	}

	task1 := task.FromCommands("t1", "true")
	task1.Context = "after_failed"

	task2 := task.FromCommands("t2", "true")
	task2.Context = "before_failed"

	err = runner.Run(task1)
	if err != nil || task1.ExitCode() != 0 {
		t.Fatal(err)
	}

	err = runner.Run(task2)
	if err == nil {
		t.Error()
	}

	if c2.StartupError() == nil || task2.ExitCode() != -1 {
		t.Error()
	}

	runner.Finish()
}

func helpSetupCleanUp() (path string, defereCleanUp func()) {
	tmpDir, _ := os.MkdirTemp(os.TempDir(), "context-envfile")
	path = filepath.Join(tmpDir, "generated_task_123.env")
	return path, func() {
		os.RemoveAll(tmpDir)
	}
}

func Test_Generate_Env_file(t *testing.T) {
	t.Run("with correctly merged output in env file from os and user supplied Env", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "var1": "userOverwrittemdd"})

		contents := genEnvFileHelperTestRunner(t, osEnvVars.Merge(userEnvVars), utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
		}))

		if strings.Contains(contents, "var1=original") {
			t.Fatal("incorrectly merged and overwritten env vars")
		}
	})
	t.Run("with forbidden variable names correctly stripped out", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222", "!::": "whatever val will never be added"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "var1": "userOverwrittemdd"})

		contents := genEnvFileHelperTestRunner(t, osEnvVars.Merge(userEnvVars), utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
		}))

		if strings.Contains(contents, "!::=whatever val will never be added") {
			t.Fatal("invalid cahrs not skipped properly and overwritten env vars")
		}
	})
	t.Run("with exclude variable names correctly stripped out", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222", "!::": "whatever val will never be added", "=::": "whatever val will never be added",
			"": "::=::", " ": "::=::", "excld1": "bye bye", "exclude3": "sadgfddf"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "var1": "userOverwrittemdd", "userSuppliedButExcluded": `¯\_(ツ)_/¯`, "UPPER_VAR_make_me_bigger": "this_key_is_large"})

		contents := genEnvFileHelperTestRunner(t, osEnvVars.Merge(userEnvVars), utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
			e.Exclude = append(e.Exclude, []string{"excld1", "exclude3", "userSuppliedButExcluded"}...)
			e.Modify = append(e.Modify, []utils.ModifyEnv{
				{Pattern: "^(?P<keyword>TF_VAR_)(?P<varname>.*)", Operation: "lower"},
				{Pattern: "^(?P<keyword>UPPER_VAR_)(?P<varname>.*)", Operation: "upper"},
			}...)
		}))

		for _, excluded := range []string{"excld1=bye bye", "exclude3=sadgfddf", `userSuppliedButExcluded=¯\_(ツ)_/¯`} {
			if slices.Contains(strings.Split(contents, "\n"), excluded) {
				t.Fatal("invalid chars not skipped properly and overwritten env vars")
			}
		}

		if slices.Contains(strings.Split(contents, "\n"), "=::=whatever val will never be added") {
			t.Fatal("invalid chars not skipped properly and overwritten env vars")
		}

		if slices.Contains(strings.Split(contents, "\n"), "!::=whatever val will never be added") {
			t.Fatal("invalid chars not skipped properly and overwritten env vars")
		}

		if !slices.Contains(strings.Split(contents, "\n"), "UPPER_VAR_MAKE_ME_BIGGER=this_key_is_large") {
			t.Fatal("Modify not changed the values properly")
		}
	})

	t.Run("with include variable names correctly set", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222", "!::": "whatever val will never be added", "=::": "whatever val will never be added 2", "incld1": "welcome var", "exclude3": "sadgfddf"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "var1": "userOverwrittemdd", "userSuppliedButExcluded": `¯\_(ツ)_/¯`})

		contents := genEnvFileHelperTestRunner(t, osEnvVars.Merge(userEnvVars), utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
			e.Exclude = append(e.Exclude, []string{}...)
			e.Include = append(e.Include, "incld1")
		}))

		for _, included := range []string{"incld1=welcome var"} {
			if !slices.Contains(strings.Split(contents, "\n"), included) {
				t.Fatal("invalid vars not skipped properly and overwritten env vars")
			}
		}
	})

	// Note about this test case
	// it will include exclude from the injected env
	// however the merging of environment variables is still case sensitive
	t.Run("with case insensitive comparison on exclude", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222", "!::": "whatever val will never be added", "=::": "whatever val will never be added 2", "incld1": "welcome var", "exclude3": "sadgfddf"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "VAR1": "userOverwrittemdd", "userSuppliedButExcluded": `¯\_(ツ)_/¯`})

		contents := genEnvFileHelperTestRunner(t, osEnvVars.Merge(userEnvVars), utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
			e.Exclude = append(e.Exclude, []string{"var1", "FOO", "UserSuppliedButEXCLUDED"}...)
		}))

		got := strings.Split(contents, "\n")
		for _, checkExcluded := range []string{"var1=original", "VAR1=userOverwrittemdd", "foo=bar", `userSuppliedButExcluded=¯\_(ツ)_/¯`} {
			if slices.Contains(got, checkExcluded) {
				t.Fatalf("invalid vars\ngot: %q\nshould have skipped ( %s )\n", got, checkExcluded)
			}
		}

		for _, checkIncluded := range []string{"var2=original222", "incld1=welcome var", "exclude3=sadgfddf"} {
			if !slices.Contains(got, checkIncluded) {
				t.Fatalf("invalid vars\ngot: %q\nshould have included ( %s )\n", got, checkIncluded)
			}
		}
	})

	t.Run("with case insensitive comparison on include", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222", "!::": "whatever val will never be added", "=::": "whatever val will never be added 2", "incld1": "welcome var", "exclude3": "sadgfddf"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "VAR1": "userOverwrittemdd", "userSuppliedButExcluded": `¯\_(ツ)_/¯`})

		contents := genEnvFileHelperTestRunner(t, osEnvVars.Merge(userEnvVars), utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
			e.Include = []string{"var1", "FOO", "UserSuppliedButEXCLUDED"}
		}))

		got := strings.Split(contents, "\n")
		for _, checkExcluded := range []string{"var1=original", "VAR1=userOverwrittemdd", "foo=bar", `userSuppliedButExcluded=¯\_(ツ)_/¯`} {
			if !slices.Contains(got, checkExcluded) {
				t.Fatalf("invalid vars\ngot: %q\nshould have skipped ( %s )\n", got, checkExcluded)
			}
		}

		for _, checkIncluded := range []string{"var2=original222", "incld1=welcome var", "exclude3=sadgfddf"} {
			if slices.Contains(got, checkIncluded) {
				t.Fatalf("invalid vars\ngot: %q\nshould have included ( %s )\n", got, checkIncluded)
			}
		}
	})

	t.Run("with include/exclude variable both set return error", func(t *testing.T) {
		outputFilePath, cleanUp := helpSetupCleanUp()

		defer cleanUp()

		osEnvVars := variables.FromMap(map[string]string{"var1": "original", "var2": "original222", "!::": "whatever val will never be added", "incld1": "welcome var", "exclude3": "sadgfddf"})
		userEnvVars := variables.FromMap(map[string]string{"foo": "bar", "var1": "userOverwrittemdd", "userSuppliedButExcluded": `¯\_(ツ)_/¯`})
		envVars := osEnvVars.Merge(userEnvVars)

		execContext := runner.NewExecutionContext(nil, "", envVars, utils.NewEnvFile(func(e *utils.Envfile) {
			e.Generate = true
			e.Path = outputFilePath
			e.Exclude = append(e.Exclude, []string{"excld1", "exclude3", "userSuppliedButExcluded"}...)
			e.Include = append(e.Include, "incld1")
		}), []string{}, []string{}, []string{}, []string{})

		if err := execContext.GenerateEnvfile(envVars); err == nil {
			t.Fatal("got nil, wanted an error")
		}

	})

}

func genEnvFileHelperTestRunner(t *testing.T, envVars *variables.Variables, envFile *utils.Envfile) string {
	t.Helper()

	execContext := runner.NewExecutionContext(nil, "", envVars, envFile, []string{}, []string{}, []string{}, []string{})

	err := execContext.GenerateEnvfile(envVars)
	if err != nil {
		t.Fatal(err)
	}

	contents, readErr := os.ReadFile(envFile.Path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(contents) < 1 {
		t.Fatal("empty file")
	}
	return string(contents)
}

func ExampleExecutionContext_GenerateEnvfile() {
	outputFilePath, cleanUp := helpSetupCleanUp()

	defer cleanUp()

	osEnvVars := variables.FromMap(map[string]string{"TF_VAR_CAPPED_BY_MSFT": "some value"})
	//  "var2": "original222", "!::": "whatever val will never be added", "incld1": "welcome var", "exclude3": "sadgfddf"})
	userEnvVars := variables.FromMap(map[string]string{})
	envVars := osEnvVars.Merge(userEnvVars)
	execContext := runner.NewExecutionContext(nil, "", envVars, utils.NewEnvFile(func(e *utils.Envfile) {
		e.Generate = true
		e.Path = outputFilePath
		e.Exclude = append(e.Exclude, []string{"excld1", "exclude3", "userSuppliedButExcluded"}...)
		e.Modify = append(e.Modify, []utils.ModifyEnv{
			{Pattern: "^(?P<keyword>TF_VAR_)(?P<varname>.*)", Operation: "lower"},
		}...)

	}), []string{}, []string{}, []string{}, []string{})

	_ = execContext.GenerateEnvfile(envVars)

	contents, _ := os.ReadFile(outputFilePath)
	// for the purposes of the test example we need to make sure the map is
	// always displayed in same order of keys, which is not a guarantee with a map
	fmt.Println(string(contents))
	//Output:
	// TF_VAR_capped_by_msft=some value
}
