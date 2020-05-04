package config

type Variables map[string]string

func NewSet(m map[string]string) Variables {
	return Variables(m)
}

func (vars *Variables) Set(name, value string) {
	if *vars == nil {
		*vars = make(map[string]string)
	}
	(*vars)[name] = value
}

func (vars Variables) Get(name string) string {
	return vars[name]
}

func (vars Variables) Has(name string) bool {
	_, ok := vars[name]
	return ok
}

func (vars Variables) Merge(src Variables) Variables {
	dst := make(Variables)
	for k, v := range vars {
		dst.Set(k, v)
	}

	for k, v := range src {
		dst.Set(k, v)
	}

	return dst
}

func (vars Variables) With(name, value string) Variables {
	dst := make(Variables)
	for k, v := range vars {
		dst[k] = v
	}

	dst[name] = value

	return dst
}
