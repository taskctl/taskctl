package config

import (
	"testing"

	"github.com/taskctl/taskctl/pkg/utils"
)

func Test_buildContext(t *testing.T) {
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
		t.Fatal()
	}

	cwd := utils.MustGetwd()
	if c.Dir != cwd {
		t.Error()
	}
}
