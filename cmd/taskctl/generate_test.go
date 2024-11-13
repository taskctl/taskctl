package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cmd "github.com/Ensono/taskctl/cmd/taskctl"
	"github.com/Ensono/taskctl/internal/utils"
)

func Test_generateCommand(t *testing.T) {

	t.Run("errors with target not set", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yml", "generate"},
			errored: true,
		})
	})
	t.Run("errors with incorrect pipeline", func(t *testing.T) {
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yml", "generate", "foo:not:there", "-t", "github"},
			errored: true,
		})
	})
	t.Run("succeeds with github implementation - with default location", func(t *testing.T) {
		pn := "graph:pipeline1"
		path := cmd.DefaultCIOutput["github"]
		testFile := filepath.Join(path, fmt.Sprintf("%s.yml", utils.ConvertToMachineFriendly(pn)))
		if err := os.MkdirAll(cmd.DefaultCIOutput["github"], 0o777); err != nil {
			t.Errorf("unable to create")
		}
		defer os.Remove(testFile)
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yml", "generate", pn, "-t", "github"},
			errored: false,
		})
	})
	t.Run("succeeds with github implementation - with custom location", func(t *testing.T) {
		pn := "graph:pipeline1"
		tmpDir, err := os.MkdirTemp("", "generate-*")
		if err != nil {
			t.Errorf("unable to create")
		}
		defer os.RemoveAll(tmpDir)
		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yml", "generate", pn, "-t", "github", "--output", tmpDir},
			errored: false,
		})
	})

	t.Run("errors with incorrect pipeline specified", func(t *testing.T) {
		pn := "graph:task2"

		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yml", "generate", pn, "-t", "github"},
			errored: true,
		})
	})
	
	t.Run("errors with incorrect target specified", func(t *testing.T) {
		pn := "graph:pipeline1"

		cmdRunTestHelper(t, &cmdRunTestInput{
			args:    []string{"-c", "testdata/generate.yml", "generate", pn, "-t", "jenkins"},
			errored: true,
		})
	})
}
