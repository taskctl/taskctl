package output

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/taskctl/taskctl/pkg/task"
)

func Test_cockpitOutputDecorator(t *testing.T) {
	b := safebuffer{}
	closeCh = make(chan bool)
	dec := newCockpitOutputWriter(&task.Task{Name: "task1"}, &b, closeCh)
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

	close(closeCh)
}

// safebuffer is a goroutine safe bytes.safebuffer
type safebuffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *safebuffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the safebuffer is a nil pointer, it returns "<nil>".
func (s *safebuffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}
