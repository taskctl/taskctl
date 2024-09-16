package output_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Ensono/taskctl/pkg/output"
	"github.com/Ensono/taskctl/pkg/task"
	"github.com/sirupsen/logrus"
)

func TestNewTaskOutput_Prefixed(t *testing.T) {
	var b bytes.Buffer
	_, err := output.NewTaskOutput(
		&task.Task{Name: "task1"},
		"unknown-format",
		&b,
		&b,
	)
	if err == nil {
		t.Error()
	}

	logrus.SetOutput(&b)
	tt := task.FromCommands("t1", "echo 1")
	tt.Name = "task1"
	o, err := output.NewTaskOutput(
		tt,
		string(output.PrefixedOutput),
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

func TestNewTaskOutput(t *testing.T) {
	var b bytes.Buffer
	_, err := output.NewTaskOutput(
		&task.Task{Name: "task1"},
		"unknown-format",
		&b,
		&b,
	)
	if err == nil {
		t.Error()
	}

	logrus.SetOutput(&b)
	tt := task.FromCommands("t1", "echo 1")
	tt.Name = "task1"
	o, err := output.NewTaskOutput(
		tt,
		string(output.RawOutput),
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
	if s != "" {
		t.Error()
	}

	_, err = o.Stdout().Write([]byte("abc"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = o.Stderr().Write([]byte("def"))
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != "abcdef" {
		t.Error()
	}

	closeCh := make(chan bool)
	to, err := output.NewTaskOutput(
		tt,
		string(output.CockpitOutput),
		&b,
		&b,
	)
	if err != nil {
		t.Fatal(err)
	}
	to.WithCloseCh(closeCh)

	to.Close()
}
