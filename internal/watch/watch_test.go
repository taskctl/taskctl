package watch

import (
	"bytes"
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

// TestWatcherRetriggers a file change must re-run the task,
// not just trigger it once. The output file lives outside the watched dir so the
// task's own writes don't feed back as events.
func TestWatcherRetriggers(t *testing.T) {
	watchDir := t.TempDir()
	trigger := filepath.Join(watchDir, "trigger.txt")
	if err := os.WriteFile(trigger, []byte{'0'}, 0644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(t.TempDir(), "runs.log")

	w, err := NewWatcher("w1", nil, []string{filepath.Join(watchDir, "*")}, nil,
		task.FromCommands("echo x >> '"+outFile+"'"))
	if err != nil {
		t.Fatal(err)
	}

	r, err := runner.NewTaskRunner()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := w.Run(r); err != nil {
			t.Error(err)
		}
	})

	deadline := time.Now().Add(5 * time.Second)
	for !w.Running() {
		if time.Now().After(deadline) {
			t.Fatal("watcher did not start running within 5 seconds")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Two writes spaced beyond the watcher's 1s poll interval.
	for i := range 2 {
		time.Sleep(1200 * time.Millisecond)
		if err := os.WriteFile(trigger, []byte{byte('a' + i)}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// A single-shot watcher records at most the one initial run; the fix must
	// yield at least one additional run from the events above.
	deadline = time.Now().Add(5 * time.Second)
	for runCount(t, outFile) < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("task ran %d time(s); expected re-trigger on file change", runCount(t, outFile))
		}
		time.Sleep(50 * time.Millisecond)
	}

	w.Close()
	wg.Wait()
}

// runCount returns how many times the watched task has run, one "x\n" per run.
func runCount(t *testing.T, outFile string) int {
	t.Helper()
	data, err := os.ReadFile(outFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return bytes.Count(data, []byte("x"))
}
