package config

type PipelineConfig struct {
	Task      string
	DependsOn interface{} `yaml:"depends_on"`
}

func (pc PipelineConfig) GetDependsOn() (deps []string) {
	return readStringsArray(pc.DependsOn)
}
