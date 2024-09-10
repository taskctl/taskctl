package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ensono/taskctl/internal/cmdutils"
)

func setupCleanUp() (dir string, deferFn func()) {
	dir, _ = os.MkdirTemp(os.TempDir(), "initTester")
	deferFn = func() {
		os.RemoveAll(dir)
	}
	return
}

func Test_initCommand(t *testing.T) {
	t.Run("custom_dir", func(t *testing.T) {
		dir, cleanUp := setupCleanUp()
		defer cleanUp()
		file := filepath.Join(dir, "tasks.yml")

		runTestHelper(t, runTestIn{
			args:   []string{"--dir", dir, "init", "tasks.yml", "--no-prompt"},
			output: []string{fmt.Sprintf(cmdutils.GREEN_TERMINAL+" "+cmdutils.MAGENTA_TERMINAL, file, "was created. Edit it accordingly to your needs")},
		})

		files, _ := os.ReadDir(dir)

		if len(files) != 1 {
			t.Fatal("Incorrect files written")
		}
	})

	t.Run("errors on missing params if not in interactive mode", func(t *testing.T) {
		dir, cleanUp := setupCleanUp()
		defer cleanUp()
		runTestHelper(t, runTestIn{
			args:    []string{"--dir", dir, "init", "--no-prompt"},
			errored: true,
		})
	})
}
