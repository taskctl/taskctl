package runner

import (
	"fmt"
)

type Variables map[string]string

func NewVariables(m map[string]string) Variables {
	variables := make(Variables)
	for k, v := range m {
		variables.Set(k, v)
	}

	return variables
}

func (vars Variables) Set(name, value string) {
	vars[name] = value
}

func (vars Variables) Get(name string) (string, bool) {
	val, ok := vars[name]
	return val, ok
}

func (vars Variables) With(name, value string) Variables {
	varc := make(Variables)
	for k, v := range vars {
		varc[k] = v
	}

	varc[name] = value

	return varc
}

func (vars Variables) Env() []string {
	var env = make([]string, 0)
	for k, v := range vars {
		env = append(env, fmt.Sprintf("VAR_%s=%s", k, v))
	}

	return env
}
