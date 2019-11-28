package builder

import (
	"github.com/trntv/wilson/pkg/util"
	"time"
)

type WilsonConfigDefinition struct {
	Shell         util.Executable
	Docker        util.Executable
	DockerCompose util.Executable
	Kubectl       util.Executable
	Ssh           util.Executable
	Debug         bool
}

type ContextDefinition struct {
	Type      string
	Dir       string
	Container ContainerDefinition
	SSH       SSHConfigDefinition
	Env       map[string]string
	Up        []string
	Down      []string
	Before    []string
	After     []string
	util.Executable
}

type StageDefinition struct {
	Name         string
	Task         string
	Pipeline     string
	DependsOn    []string
	Env          map[string]string
	AllowFailure bool
}

type TaskDefinition struct {
	Name    string
	Command []string
	Context string
	Env     map[string]string
	Dir     string
	Timeout *time.Duration
}

type WatcherDefinition struct {
	Events []string
	Watch  []string
	Task   string
}

type ContainerDefinition struct {
	Provider string
	Name     string
	Image    string
	Exec     bool
	Options  []string
	Env      map[string]string
	util.Executable
}

type SSHConfigDefinition struct {
	Options []string
	User    string
	Host    string
	util.Executable
}
