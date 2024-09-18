package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
)

func TestConfig_decode(t *testing.T) {
	loader := config.NewConfigLoader(config.NewConfig())
	loader.WithStrictDecoder()
	cwd, _ := os.Getwd()
	def, err := loader.Load(filepath.Join(cwd, "testdata", "tasks.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := def.Tasks["test-task"]; !ok {
		t.Fatal("tasks parsing error")
	}

	if _, ok := def.Pipelines["pipeline2"]; !ok {
		t.Fatal("pipelines parsing error")
	}

	if len(def.Pipelines) != 2 {
		t.Fatal("pipelines parsing failed")
	}
}

// TODO:  config.merge is tested implicitly
// needs to provide negative cases which throw
// config merge
func TestNewConfig(t *testing.T) {
	cfg := config.NewConfig()
	if !cfg.Variables.Has("TempDir") {
		t.Error()
	}
}

func TestConfig_Errors_onContainer(t *testing.T) {
	_ = os.MkdirAll(".taskctl-tester", 0777)
	defer os.RemoveAll(".taskctl-tester")
	os.WriteFile(filepath.Join(".taskctl-tester", "incorrect-container.yaml"), []byte(`
contexts:
  test:
    container:
      shell: foo
    envfile:
      exclude:
        - SOURCEVERSIONMESSAGE
        - JAVA
        - GO
        - HOMEBREW
`), 0777)

	loader := config.NewConfigLoader(config.NewConfig())
	loader.WithStrictDecoder()
	cwd, _ := os.Getwd()
	_, err := loader.Load(filepath.Join(cwd, filepath.Join(".taskctl-tester", "incorrect-container.yaml")))
	if err == nil {
		t.Fatal(err)
	}

	if !errors.Is(err, config.ErrBuildContextIncorrect) {
		t.Fatalf("wrong error\n\ngot: %v\nwanted: %v", err, config.ErrBuildContextIncorrect)
	}
}

func TestConfig_Errors_onExecutable(t *testing.T) {
	_ = os.MkdirAll(".taskctl-tester", 0777)
	defer os.RemoveAll(".taskctl-tester")
	os.WriteFile(filepath.Join(".taskctl-tester", "incorrect-exec.yaml"), []byte(`
contexts:
  test:
    executable:
      args: ["foo"]
    envfile:
      exclude:
        - SOURCEVERSIONMESSAGE
        - JAVA
        - GO
        - HOMEBREW
`), 0777)

	loader := config.NewConfigLoader(config.NewConfig())
	loader.WithStrictDecoder()
	cwd, _ := os.Getwd()
	_, err := loader.Load(filepath.Join(cwd, filepath.Join(".taskctl-tester", "incorrect-exec.yaml")))
	if err == nil {
		t.Fatal(err)
	}

	if !errors.Is(err, config.ErrBuildContextIncorrect) {
		t.Fatalf("wrong error\n\ngot: %v\nwanted: %v", err, config.ErrBuildContextIncorrect)
	}
}
