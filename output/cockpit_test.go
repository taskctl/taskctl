package output

import (
	"bytes"
	"testing"

	"github.com/taskctl/taskctl/task"
)

func Test_cockpitOutputDecorator(t *testing.T) {
	b := bytes.NewBuffer([]byte{})
	closeCh = make(chan bool)
	dec := newCockpitOutputWriter(&task.Task{Name: "task1"}, b, closeCh)
	err := dec.WriteHeader()
	if err != nil {
		t.Fatal(err)
	}

	n, err := dec.Write([]byte("lorem ipsum"))
	if err != nil && n == 0 {
		t.Fatal()
	}

	err = dec.WriteFooter()
	if err != nil {
		t.Fatal(err)
	}

	close(closeCh)
}
