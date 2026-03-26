package variables

import (
	"sync"
)

// Container is an interface of variables container.
// Is is simple key-value structure.
type Container interface {
	Set(string, any)
	Get(string) any
	Has(string) bool
	Map() map[string]any
	Merge(Container) Container
	With(string, any) Container
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
func (vars *Variables) Set(key string, value any) {
	vars.m.Store(key, value)
}

// Get returns value by given key
func (vars *Variables) Get(key string) any {
	v, ok := vars.m.Load(key)
	if !ok {
		return ""
	}

	return v
}

// Has checks if value exists
func (vars *Variables) Has(name string) bool {
	_, ok := vars.m.Load(name)
	return ok
}

// Map returns container in map[string]string form
func (vars *Variables) Map() map[string]any {
	m := make(map[string]any)
	vars.m.Range(func(key, value any) bool {
		m[key.(string)] = value
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
func (vars *Variables) With(key string, value any) Container {
	dst := &Variables{}
	dst = dst.Merge(vars).(*Variables)
	dst.Set(key, value)

	return dst
}
