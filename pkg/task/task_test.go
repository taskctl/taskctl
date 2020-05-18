package task

import "testing"

func TestNewTask(t *testing.T) {
	task := FromCommands("ls /tmp")
	task.WithEnv("TEST_ENV", "TEST_VAL")

	if task.Commands[0] != "ls /tmp" {
		t.Error("task creation failed")
	}

	if task.Env.Get("TEST_ENV") != "TEST_VAL" {
		t.Error("task's env creation failed")
	}
}
