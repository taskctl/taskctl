package config

type PipelineConfig struct {
	Task    string
	Depends interface{}
}

func (pc PipelineConfig) DependsOn() (deps []string) {
	return readStringsArray(pc.Depends)
}
