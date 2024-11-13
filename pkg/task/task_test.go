package task

import (
	"fmt"
	"testing"
)

func TestTask(t *testing.T) {
	task := FromCommands("t1", "ls /tmp")
	task.WithEnv("TEST_ENV", "TEST_VAL")

	if task.Commands[0] != "ls /tmp" {
		t.Error("task creation failed")
	}

	if task.Env.Get("TEST_ENV") != "TEST_VAL" {
		t.Error("task's env creation failed")
	}

	if task.Duration().Seconds() <= 0 {
		t.Error()
	}
}

func TestTask_ErrorMessage(t *testing.T) {
	task := NewTask("abc")
	task.WithError(fmt.Errorf("true"))

	if task.ErrorMessage() != "true" {
		t.Error()
	}

	task = NewTask("errored")
	if task.ErrorMessage() != "" {
		t.Error()
	}

	task.WithError(fmt.Errorf("true"))
	if task.Error().Error() != "true" {
		t.Error()
	}
}

func TestNewTask_WithVariations(t *testing.T) {
	task := FromCommands("t1", "ls /tmp")

	if len(task.GetVariations()) != 1 {
		t.Error()
	}

	task.Variations = []map[string]string{{"GOOS": "linux"}, {"GOOS": "windows"}}
	if len(task.GetVariations()) != 2 {
		t.Error()
	}
}
