package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/taskctl/taskctl/pkg/task"
)

func Test_cockpitOutputDecorator(t *testing.T) {
	var b bytes.Buffer
	dec := newCockpitOutputWriter(&task.Task{Name: "task1"}, &b)
	err := dec.WriteHeader()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(frame * 2)

	if !strings.Contains(b.String(), "Running: task1") {
		t.Fatal()
	}

	n, err := dec.Write([]byte("lorem ipsum"))
	if err != nil && n == 0 {
		t.Fatal()
	}

	if strings.Contains(b.String(), "lorem ipsum") {
		t.Fatal()
	}

	err = dec.WriteFooter()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(frame * 2)

	if !strings.Contains(b.String(), "Finished") {
		t.Fatal()
	}

	close(base.closeCh)
}
