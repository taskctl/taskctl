package config

type PipelineConfig struct {
	Task    string
	Depends interface{}
}

func (pc PipelineConfig) DependsOn() (deps []string) {
	if pc.Depends == nil {
		return deps
	}

	deps, ok := pc.Depends.([]string)
	if !ok {
		dep, ok := pc.Depends.(string)
		if ok {
			deps = []string{dep}
		}

	}

	return deps
}
