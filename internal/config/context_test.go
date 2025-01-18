package config

import (
	"testing"

	"github.com/taskctl/taskctl/variables"

	"github.com/taskctl/taskctl/utils"
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

	cwd := utils.MustGetwd()
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
