package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taskctl/taskctl/internal/fsutil"
	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/variables"
)

func Test_buildContext_dir(t *testing.T) {
	c, err := buildContext(&contextDefinition{
		Up:        []string{"true"},
		Down:      []string{"true"},
		Before:    []string{"true"},
		After:     []string{"true"},
		Env:       map[string]string{},
		Variables: map[string]string{},
		Quote:     "'",
	})
	if err != nil {
		t.Fatal(err)
	}

	cwd := fsutil.MustGetwd()
	if c.Dir != cwd {
		t.Error()
	}
}

func Test_buildContext_env_file(t *testing.T) {
	c, err := buildContext(&contextDefinition{
		Env:       map[string]string{},
		EnvFile:   "testdata/.env",
		Variables: map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range variables.FromMap(map[string]string{"VAR_1": "VAL_1_2", "VAR_2": "VAL_2"}).Map() {
		if c.Env.Get(k) != v {
			t.Errorf("buildContext() env error, want %s, got %s", v, c.Env.Get(k))
		}
	}
}

func Test_buildContext_typed_contexts_from_config(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cl := NewConfigLoader(NewConfig())
	cfg, err := cl.Load(filepath.Join(cwd, "testdata", "typed_contexts.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	local := cfg.Contexts["local_ctx"]
	if local == nil {
		t.Fatal("local_ctx not found")
	}
	if local.Executable == nil {
		t.Error("local_ctx: expected non-nil Executable")
	}
	if local.Dir != "/tmp" {
		t.Errorf("local_ctx: got dir %q", local.Dir)
	}

	cases := []string{"docker_run_ctx", "docker_exec_ctx", "kubernetes_ctx", "ssh_ctx"}
	for _, name := range cases {
		c := cfg.Contexts[name]
		if c == nil {
			t.Fatalf("%s not found", name)
		}
		if c.Executable != nil {
			t.Errorf("%s: expected nil Executable, got %+v", name, c.Executable)
		}
	}

	dockerRun := cfg.Contexts["docker_run_ctx"]
	if dockerRun.Dir != "/tmp" {
		t.Errorf("docker_run_ctx: got dir %q", dockerRun.Dir)
	}
	if dockerRun.Env.Get("FOO") != "bar" {
		t.Errorf("docker_run_ctx: got env FOO=%q", dockerRun.Env.Get("FOO"))
	}

	sshCtx := cfg.Contexts["ssh_ctx"]
	if sshCtx.Dir != "/tmp" {
		t.Errorf("ssh_ctx: got dir %q", sshCtx.Dir)
	}
}

func Test_buildContext_validation_errors(t *testing.T) {
	tests := []struct {
		name string
		def  *contextDefinition
		want string
	}{
		{
			name: "unknown type",
			def:  &contextDefinition{Type: "vagrant"},
			want: "unknown context type",
		},
		{
			name: "docker neither image nor container",
			def:  &contextDefinition{Type: "docker", Docker: &dockerDefinition{}},
			want: "exactly one of image or container",
		},
		{
			name: "docker both image and container",
			def:  &contextDefinition{Type: "docker", Docker: &dockerDefinition{Image: "alpine", Container: "c1"}},
			want: "exactly one of image or container",
		},
		{
			name: "docker missing block",
			def:  &contextDefinition{Type: "docker"},
			want: "requires a docker block",
		},
		{
			name: "kubernetes without pod",
			def:  &contextDefinition{Type: "kubernetes", Kubernetes: &kubernetesDefinition{}},
			want: "requires pod",
		},
		{
			name: "ssh without host",
			def:  &contextDefinition{Type: "ssh", SSH: &sshDefinition{}},
			want: "requires host",
		},
		{
			name: "typed context also sets executable",
			def: &contextDefinition{
				Type:       "docker",
				Docker:     &dockerDefinition{Image: "alpine"},
				Executable: runner.Binary{Bin: "bash"},
			},
			want: "does not accept an executable block",
		},
		{
			name: "local context sets docker block",
			def:  &contextDefinition{Docker: &dockerDefinition{Image: "alpine"}},
			want: "does not accept a docker block",
		},
		{
			name: "docker type also sets ssh block",
			def: &contextDefinition{
				Type:   "docker",
				Docker: &dockerDefinition{Image: "alpine"},
				SSH:    &sshDefinition{Host: "example.com"},
			},
			want: "does not accept a ssh block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildContext(tt.def)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("got error %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}
