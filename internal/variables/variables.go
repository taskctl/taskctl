package variables

import (
	"sync"

	"github.com/taskctl/taskctl/internal/utils"
)

type Container interface {
	Set(string, string)
	Get(string) string
	Map() map[string]string
	Merge(Container) Container
	With(string, string) Container
}

type Variables struct {
	m sync.Map
}

func NewVariables(values map[string]string) *Variables {
	vars := &Variables{}
	for k, v := range values {
		vars.m.Store(k, v)
	}

	return vars
}

func NewVariablesFromEnv(env []string) *Variables {
	m := utils.ParseEnv(env)
	return NewVariables(m)
}

func (vars *Variables) Set(name, value string) {
	vars.m.Store(name, value)
}

func (vars *Variables) Get(name string) string {
	v, ok := vars.m.Load(name)
	if !ok {
		return ""
	}

	return v.(string)
}

func (vars *Variables) Has(name string) bool {
	_, ok := vars.m.Load(name)
	return ok
}

func (vars *Variables) Map() map[string]string {
	m := make(map[string]string)
	vars.m.Range(func(key, value interface{}) bool {
		m[key.(string)] = value.(string)
		return true
	})
	return m
}

func (vars *Variables) Merge(src Container) Container {
	dst := &Variables{}

	if vars != nil {
		for k, v := range vars.Map() {
			dst.Set(k, v)
		}
	}

	for k, v := range src.Map() {
		dst.Set(k, v)
	}

	return dst
}

func (vars *Variables) With(name, value string) Container {
	dst := &Variables{}
	dst = dst.Merge(vars).(*Variables)
	dst.Set(name, value)

	return dst
}
