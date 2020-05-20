package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/taskctl/taskctl/pkg/task"
)

func Test_prefixedOutputDecorator(t *testing.T) {
	var l, b bytes.Buffer
	logrus.SetOutput(&l)

	dec := newPrefixedOutputWriter(&task.Task{Name: "task1"}, &b)
	err := dec.WriteHeader()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(l.String(), "Running task task1...") {
		t.Fatal()
	}

	n, err := dec.Write([]byte("lorem ipsum"))
	if err != nil && n == 0 {
		t.Fatal()
	}

	if !strings.Contains(b.String(), "task1") || !strings.Contains(b.String(), "lorem ipsum") {
		t.Fatal()
	}

	err = dec.WriteFooter()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(l.String(), "task1 finished") {
		t.Fatal()
	}
}
