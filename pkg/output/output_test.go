package output

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"github.com/taskctl/taskctl/pkg/task"
	"strings"
	"testing"
)

func TestNewTaskOutput(t *testing.T) {
	var b bytes.Buffer
	_, err := NewTaskOutput(
		&task.Task{Name: "task1"},
		"unknown-format",
		&b,
		&b,
	)
	if err == nil {
		t.Error()
	}

	logrus.SetOutput(&b)
	tt := task.FromCommands("echo 1")
	tt.Name = "task1"
	o, err := NewTaskOutput(
		tt,
		FormatPrefixed,
		&b,
		&b,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = o.Start()
	if err != nil {
		t.Fatal(err)
	}

	err = o.Finish()
	if err != nil {
		t.Fatal(err)
	}

	s := b.String()
	if !strings.Contains(s, "Running") || !strings.Contains(s, "finished") || !strings.Contains(s, "Duration") {
		t.Error()
	}
}
