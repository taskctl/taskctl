package watch

import (
	"github.com/taskctl/taskctl/pkg/task"
	"os"
	"testing"
)

func TestNewWatcher(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWatcher("w1", []string{eventCreate}, []string{cwd}, []string{}, task.FromCommands("true"))
	if err != nil {
		t.Fatal(err)
	}

	w.Close()
}
