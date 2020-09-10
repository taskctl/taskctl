package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const sampleCfg = "{\"tasks\": {\"task1\": {\"command\": [\"true\"]}}}"

func TestLoader_Load(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := NewConfigLoader(NewConfig())
	cfg, err := cl.Load(filepath.Join(cwd, "testdata", "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	if cfg.Contexts["local_wth_quote"].Quote != "'" {
		t.Error("context's quote parsing failed")
	}

	cl = NewConfigLoader(NewConfig())
	cl.dir = filepath.Join(cwd, "testdata")
	cfg, err = cl.Load("test.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	cl = NewConfigLoader(NewConfig())
	cl.dir = filepath.Join(cwd, "testdata", "nested")
	cfg, err = cl.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Tasks["test-task"]; !ok {
		t.Error("yaml parsing failed")
	}

	_, err = cl.LoadGlobalConfig()
	if err != nil {
		t.Fatal()
	}
}

func TestLoader_resolveDefaultConfigFile(t *testing.T) {
	cl := NewConfigLoader(NewConfig())

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

func TestLoader_loadDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := NewConfigLoader(NewConfig())
	m, err := cl.loadDir(filepath.Join(cwd, "testdata"))
	if err != nil {
		t.Fatal(err)
	}

	tasks := m["tasks"].(map[interface{}]interface{})
	if len(tasks) != 5 {
		t.Error()
	}
}

func TestLoader_readURL(t *testing.T) {
	var r int
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "")
		if r == 0 {
			writer.Header().Set("Content-Type", "application/json")
		}
		if r == 2 {
			writer.WriteHeader(500)
		}
		fmt.Fprintln(writer, sampleCfg)
		r++
	}))

	cl := NewConfigLoader(NewConfig())
	m, err := cl.readURL(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	tasks := m["tasks"].(map[string]interface{})
	if len(tasks) != 1 {
		t.Error()
	}

	_, err = cl.readURL(srv.URL)
	if err != nil {
		t.Fatal()
	}

	_, err = cl.readURL(srv.URL)
	if err == nil {
		t.Fatal()
	}
}

func TestLoader_LoadGlobalConfig(t *testing.T) {
	h := os.TempDir()
	_ = os.RemoveAll(filepath.Join(h, ".taskctl"))
	err := os.Mkdir(filepath.Join(h, ".taskctl"), 0744)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(h, ".taskctl", "config.yaml"), []byte(sampleCfg), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cl := NewConfigLoader(NewConfig())
	cl.homeDir = h
	cfg, err := cl.LoadGlobalConfig()
	if err != nil {
		t.Fatal()
	}

	if len(cfg.Tasks) == 0 {
		t.Error()
	}
}

func TestLoader_unmarshalData(t *testing.T) {
	cl := NewConfigLoader(NewConfig())
	_, err := cl.unmarshalData([]byte(sampleCfg), ".json")
	if err != nil {
		t.Error(err)
	}

	_, err = cl.unmarshalData([]byte(sampleCfg), ".txt")
	if err == nil {
		t.Error()
	}
}
