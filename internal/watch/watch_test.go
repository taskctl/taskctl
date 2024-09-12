package watch

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fsnotify/fsnotify"

	"github.com/Ensono/taskctl/pkg/runner"
	"github.com/Ensono/taskctl/pkg/task"
)

func TestNewWatcher(t *testing.T) {
	t.Skip()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWatcher("w1", []string{}, []string{filepath.Join(cwd, "*")}, []string{"watch_test.go"}, task.FromCommands("t1", "true"))
	if err != nil {
		t.Fatal(err)
	}

	r, err := runner.NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = w.Run(r)
		if err != nil {
			t.Error(err)
		}
	}()

	w.fsw.Events <- fsnotify.Event{
		Name: "fake_file.json",
		Op:   fsnotify.Rename,
	}

	w.Close()
	wg.Wait()
}
