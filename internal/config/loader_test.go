package config_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Ensono/taskctl/internal/config"
)

var sampleCfg = []byte(`{"tasks": {"task1": {"command": ["true"]}}}`)

func TestLoader_Load(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
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

	cl = config.NewConfigLoader(config.NewConfig())
	cl.WithDir(filepath.Join(cwd, "testdata"))
	cfg, err = cl.Load("test.toml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tasks["task1"] == nil || cfg.Tasks["task1"].Commands[0] != "echo true" {
		t.Error("yaml parsing failed")
	}

	cl = config.NewConfigLoader(config.NewConfig())
	cl.WithDir(filepath.Join(cwd, "testdata", "nested"))
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
	cl := config.NewConfigLoader(config.NewConfig())
	cl.WithDir(filepath.Join(cl.Dir(), "testdata"))

	file, err := cl.ResolveDefaultConfigFile()
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(file) != "tasks.yaml" {
		t.Error()
	}

	cl.WithDir("/")
	file, err = cl.ResolveDefaultConfigFile()
	if err == nil || file != "" {
		t.Error()
	}
}

func TestLoader_LoadDirImport(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	conf, err := cl.Load(filepath.Join(cwd, "testdata", "dir-dep-import.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if len(conf.Tasks) != 5 {
		t.Error()
	}
}

func TestLoader_ReadConfigFromURL(t *testing.T) {
	ttests := map[string]struct {
		contentType    string
		responseBytes  []byte
		wantError      bool
		taskCount      int
		additionalPath string
	}{
		"correct json": {
			"application/json",
			sampleCfg, false, 1, "",
		},
		"correct json from file": {
			"application/x-unknown",
			sampleCfg, false, 1, "/config.json",
		},
		"correct toml": {
			"application/toml",
			[]byte(`[tasks.task1]
command = [ true ]
`),
			false, 1, ""},
	}
	for name, tt := range ttests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", tt.contentType)
				_, err := writer.Write([]byte(tt.responseBytes))
				if err != nil {
					t.Errorf("failed to write bytes to response stream")
				}
			}))

			cl := config.NewConfigLoader(config.NewConfig())
			// cl.WithStrictDecoder()
			config, err := cl.Load(srv.URL + tt.additionalPath)
			if err != nil && !tt.wantError {
				t.Error("got error, wanted nil")
			}
			if tt.wantError && err == nil {
				t.Error("got nil, wanted error")
			}

			if len(config.Tasks) != tt.taskCount {
				t.Errorf("got %v count, wanted %v task count", len(config.Tasks), tt.taskCount)
			}
		})
	}

	// yaml needs to be run separately "¯\_(ツ)_/¯"
	t.Run("yaml parsed correctly", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/x-yaml")
			_, err := writer.Write([]byte(`
tasks:
  task1:
    command:
      - true
`))
			if err != nil {
				t.Errorf("failed to write bytes to response stream")
			}
		}))

		cl := config.NewConfigLoader(config.NewConfig())
		m, err := cl.Load(srv.URL)
		if err != nil {
			t.Fatal("got error, wanted nil")
		}
		if len(m.Tasks) != 1 {
			t.Errorf("got %v count, wanted %v task count", len(m.Tasks), 1)
		}
	})
}

func TestLoader_errors(t *testing.T) {
	t.Run("on failed status code", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(500)
		}))
		cl := config.NewConfigLoader(config.NewConfig())
		_, err := cl.Load(srv.URL)
		if err == nil {
			t.Fatal("got nil, wanted error")
		}
	})

	t.Run("on unable to figure out mediaType", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "")
			writer.Write(sampleCfg)
		}))
		cl := config.NewConfigLoader(config.NewConfig())
		_, err := cl.Load(srv.URL)
		if err == nil {
			t.Fatal("got nil, wanted error")
		}
	})
}

func TestLoader_LoadGlobalConfig(t *testing.T) {
	h := os.TempDir()
	originalHomeNix, originalHomeWin := os.Getenv("HOME"), os.Getenv("USERPROFILE")
	os.Setenv("HOME", h)
	// windows...
	os.Setenv("USERPROFILE", h)

	defer func() {
		_ = os.RemoveAll(filepath.Join(h, ".taskctl"))
		os.Setenv("HOME", originalHomeNix)
		// windows...
		os.Setenv("USERPROFILE", originalHomeWin)
	}()

	err := os.Mkdir(filepath.Join(h, ".taskctl"), 0744)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(h, ".taskctl", "config.yaml"), []byte(sampleCfg), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cl := config.NewConfigLoader(config.NewConfig())
	// cl.homeDir = h
	cfg, err := cl.LoadGlobalConfig()
	if err != nil {
		t.Fatal()
	}

	if len(cfg.Tasks) == 0 {
		t.Error()
	}
}

func TestLoader_contexts(t *testing.T) {
	dir, _ := os.MkdirTemp(os.TempDir(), "context*")
	fname := filepath.Join(dir, "context.yaml")

	f, _ := os.Create(fname)
	defer os.RemoveAll(dir)
	f.Write([]byte(`contexts:
  docker:context:
    executable:
      bin: docker
      args:
        - "run"
        - "--rm"
        - "alpine"
        - "sh"
        - "-c"
    quote: "'"
    envfile:
      generate: true
      exclude: 
        - PATH
  powershell:
    container:
      name: ensono/eir-infrastructure:1.1.251
      shell: pwsh
      shell_args:
        - -NonInteractive
        - -Command
      container_args: []
    envfile:
      exclude:
        - SOURCEVERSIONMESSAGE
        - JAVA
        - GO
        - HOMEBREW
  dind:
    container:
      name: ensono/eir-infrastructure:1.1.251
      enable_dind: true
      entrypoint: "/usr/bin/env"
      shell: bash
      shell_args:
        - -c
      container_args: []
    envfile:
      exclude:
        - SOURCEVERSIONMESSAGE
        - JAVA
        - GO
        - HOMEBREW
`))
	loader := config.NewConfigLoader(config.NewConfig())
	loader.WithStrictDecoder()
	def, err := loader.Load(fname)
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Contexts) != 3 {
		t.Errorf("got: %v\nwanted: 3\n", len(def.Contexts))
	}
	pwshContainer, ok := def.Contexts["powershell"]
	if !ok {
		t.Errorf("powershell context not found")
	}
	dindContainer, ok := def.Contexts["dind"]
	if !ok {
		t.Errorf("dind context not found")
	}

	oldDockerContext, ok := def.Contexts["docker:context"]
	if !ok {
		t.Errorf("powershell context not found")
	}

	if !pwshContainer.Executable.IsContainer {
		t.Errorf("\npwshContainer IsContainer not correctly processed\n\ngot: %v\nwanted: false", pwshContainer.Executable.IsContainer)
	}

	if !dindContainer.Executable.IsContainer {
		t.Errorf("\ndindContainer IsContainer not correctly processed\n\ngot: %v\nwanted: false", dindContainer.Executable.IsContainer)
	}

	if oldDockerContext.Executable.IsContainer {
		t.Errorf("\noldDockerContext IsContainer not correctly processed\n\ngot: %v\nwanted: false", oldDockerContext.Executable.IsContainer)
	}
	dindArgs := dindContainer.Executable.BuildArgsWithEnvFile("some-file.env")
	wantDindArgs := []string{"run", "--rm", "--env-file", "some-file.env", "-v", "${PWD}:/workspace/.taskctl", "--entrypoint", "/usr/bin/env",
		"-v", "/var/run/docker.sock:/var/run/docker.sock", "-w", "/workspace/.taskctl", "ensono/eir-infrastructure:1.1.251", "bash", "-c"}
	if !slices.Equal(dindArgs, wantDindArgs) {
		t.Errorf("dindContainer incorrectly parsed args: %v", dindArgs)
	}
}
