package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_Load(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := NewConfigLoader()
	cfg, err := cl.Load(filepath.Join(cwd, "testdata", "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	cl = NewConfigLoader()
	cfg, err = cl.Load(filepath.Join(cwd, "testdata", "test.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	_, err = cl.LoadGlobalConfig()
	if err != nil {
		t.Fatal()
	}
}

func TestLoader_resolveDefaultConfigFile(t *testing.T) {
	cl := NewConfigLoader()

	cl.dir = filepath.Join(cl.dir, "testdata")
	file, err := cl.resolveDefaultConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(file) != "tasks.yaml" {
		t.Error()
	}

	cl.dir = "/"
	file, err = cl.resolveDefaultConfigFile()
	if err == nil || file != "" {
		t.Error()
	}
}
