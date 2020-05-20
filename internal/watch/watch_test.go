package watch

import (
	"github.com/taskctl/taskctl/pkg/task"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWatcher(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewWatcher("w1", []string{eventCreate}, []string{filepath.Join(cwd, "*.exe")}, []string{}, task.FromCommands("true"))
	if err != nil {
		t.Fatal(err)
	}
}
