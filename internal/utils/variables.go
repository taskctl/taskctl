package utils

import "sync"

type Variables struct {
	m sync.Map
}

func NewVariables(m map[string]string) *Variables {
	vars := &Variables{}
	for k, v := range m {
		vars.m.Store(k, v)
	}

	return vars
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

func (vars *Variables) Merge(src *Variables) *Variables {
	dst := &Variables{}

	for k, v := range vars.Map() {
		dst.Set(k, v)
	}

	for k, v := range src.Map() {
		dst.Set(k, v)
	}

	return dst
}

func (vars *Variables) With(name, value string) *Variables {
	dst := &Variables{}
	dst.Merge(vars)
	dst.Set(name, value)

	return dst
}
