package config

import (
	"github.com/taskctl/taskctl/internal/util"
)

type Set map[string]string

func NewSet(m map[string]string) Set {
	return Set(m)
}

func (vars *Set) Set(name, value string) {
	if *vars == nil {
		*vars = make(map[string]string)
	}
	(*vars)[name] = value
}

func (vars Set) Get(name string) string {
	return vars[name]
}

func (vars Set) Has(name string) bool {
	_, ok := vars[name]
	return ok
}

func (vars Set) Merge(src Set) Set {
	dst := make(Set)
	for k, v := range vars {
		dst.Set(k, v)
	}

	for k, v := range src {
		dst.Set(k, v)
	}

	return dst
}

func (vars Set) With(name, value string) Set {
	dst := make(Set)
	for k, v := range vars {
		dst[k] = v
	}

	dst[name] = value

	return dst
}

func (vars Set) Env() []string {
	return util.ConvertEnv(vars)
}
