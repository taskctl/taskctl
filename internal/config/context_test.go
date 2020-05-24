package config

import (
	"os"
	"testing"
)

func Test_buildContext(t *testing.T) {
	c, err := buildContext(&contextDefinition{
		Up:        []string{"true"},
		Down:      []string{"true"},
		Before:    []string{"true"},
		After:     []string{"true"},
		Env:       map[string]string{},
		Variables: map[string]string{},
	})
	if err != nil {
		t.Fatal()
	}

	cwd, _ := os.Getwd()
	if c.Dir != cwd {
		t.Error()
	}
}
