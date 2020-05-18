package variables

import (
	"sync"
)

// Container is an interface of variables container.
// Is is simple key-value structure.
type Container interface {
	Set(string, string)
	Get(string) string
	Has(string) bool
	Map() map[string]string
	Merge(Container) Container
	With(string, string) Container
}

// Variables is struct containing simple key-value string values
type Variables struct {
	m sync.Map
}

// NewVariables creates new Variables instance
func NewVariables() *Variables {
	return &Variables{}
}

// FromMap creates new Variables instance from given map
func FromMap(values map[string]string) Container {
	vars := &Variables{}
	for k, v := range values {
		vars.m.Store(k, v)
	}

	return vars
}

// Set stores value with given key
func (vars *Variables) Set(key, value string) {
	vars.m.Store(key, value)
}

// Get returns value by given key
func (vars *Variables) Get(key string) string {
	v, ok := vars.m.Load(key)
	if !ok {
		return ""
	}

	return v.(string)
}

// Has checks if value exists
func (vars *Variables) Has(name string) bool {
	_, ok := vars.m.Load(name)
	return ok
}

// Map returns container in map[string]string form
func (vars *Variables) Map() map[string]string {
	m := make(map[string]string)
	vars.m.Range(func(key, value interface{}) bool {
		m[key.(string)] = value.(string)
		return true
	})
	return m
}

// Merge merges two Containers into new one
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

// With creates new container and sets key to given value
func (vars *Variables) With(key, value string) Container {
	dst := &Variables{}
	dst = dst.Merge(vars).(*Variables)
	dst.Set(key, value)

	return dst
}
