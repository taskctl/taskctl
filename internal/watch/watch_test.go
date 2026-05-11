package watch

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/task"
)

func TestNewWatcher(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = os.Remove("fake_file.json")
	})

	w, err := NewWatcher("w1", []string{}, []string{filepath.Join(cwd, "*")}, []string{"watch_test.go"}, task.FromCommands("true"))
	if err != nil {
		t.Fatal(err)
	}

	r, err := runner.NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		err := w.Run(r)
		if err != nil {
			t.Error(err)
		}
	})

	err = os.WriteFile(filepath.Join(cwd, "fake_file.json"), []byte{}, 0644)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for !w.Running() {
		if time.Now().After(deadline) {
			t.Fatal("watcher did not start running within 5 seconds")
		}
		time.Sleep(10 * time.Millisecond)
	}

	w.Close()
	wg.Wait()
}
