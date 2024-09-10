package config_test

import (
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

// TODO:  config.merge is tested implicitely
// needs to provide negative cases which throw
// config merge

func TestNewConfig(t *testing.T) {
	cfg := config.NewConfig()
	if !cfg.Variables.Has("TempDir") {
		t.Error()
	}
}
