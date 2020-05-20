package config

import (
	"testing"
)

func Test_buildContext(t *testing.T) {
	c, err := buildContext(&contextDefinition{
		Dir:       "/tmp",
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

	if c.Dir != "/tmp" {
		t.Error()
	}
}
