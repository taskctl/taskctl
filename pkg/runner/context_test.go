package runner

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/task"
	"github.com/taskctl/taskctl/pkg/variables"
)

func TestContext(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)

	c1 := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"true"}, []string{"false"}, []string{"true"}, []string{"false"})
	c2 := NewExecutionContext(nil, "/", variables.NewVariables(), []string{"false"}, []string{"false"}, []string{"false"}, []string{"false"})

	runner, err := NewTaskRunner(WithContexts(map[string]*ExecutionContext{"after_failed": c1, "before_failed": c2}))
	if err != nil {
		t.Fatal(err)
	}

	task1 := task.FromCommands("true")
	task1.Context = "after_failed"

	task2 := task.FromCommands("true")
	task2.Context = "before_failed"

	err = runner.Run(task1)
	if err != nil || task1.ExitCode != 0 {
		t.Fatal(err)
	}

	err = runner.Run(task2)
	if err == nil {
		t.Error()
	}

	if c2.startupError == nil || task2.ExitCode != -1 {
		t.Error()
	}

	runner.Finish()
}
