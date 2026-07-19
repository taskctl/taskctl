package config

import (
	"fmt"
	"path/filepath"

	"github.com/taskctl/taskctl/internal/envutil"
	"github.com/taskctl/taskctl/internal/fsutil"
	"github.com/taskctl/taskctl/runner"
	"github.com/taskctl/taskctl/variables"
)

type contextDefinition struct {
	Dir        string
	Up         []string
	Down       []string
	Before     []string
	After      []string
	Env        map[string]string
	EnvFile    string `mapstructure:"env_file"`
	Variables  map[string]string
	Executable runner.Binary
	Quote      string
	Type       string
	Docker     *dockerDefinition
	Kubernetes *kubernetesDefinition
	SSH        *sshDefinition
}

type dockerDefinition struct {
	Image     string
	Container string
	Host      string
	Options   []string
	Shell     []string
}

type kubernetesDefinition struct {
	Pod         string
	Namespace   string
	Container   string
	KubeContext string `mapstructure:"kube_context"`
	Kubeconfig  string
	Options     []string
	Shell       []string
}

type sshDefinition struct {
	Host         string
	User         string
	Port         int
	IdentityFile string `mapstructure:"identity_file"`
	Options      []string
}

func buildContext(def *contextDefinition) (*runner.ExecutionContext, error) {
	dir := def.Dir
	if dir == "" {
		dir = fsutil.MustGetwd()
	}

	envs := variables.FromMap(def.Env)
	if def.EnvFile != "" {
		filename := def.EnvFile
		if !filepath.IsAbs(filename) && dir != "" {
			filename = filepath.Join(dir, filename)
		}

		fileEnvs, err := envutil.ReadEnvFile(filename)
		if err != nil {
			return nil, err
		}

		envs = variables.FromMap(fileEnvs).Merge(envs)
	}

	executable, opts, err := resolveContextType(def)
	if err != nil {
		return nil, err
	}
	opts = append(opts, runner.WithQuote(def.Quote))

	c := runner.NewExecutionContext(
		executable,
		dir,
		envs,
		def.Up,
		def.Down,
		def.Before,
		def.After,
		opts...,
	)
	c.Variables = variables.FromMap(def.Variables)

	return c, nil
}

// resolveContextType validates def.Type against its nested type-specific blocks
// and returns the executable (non-nil only for the local/escape-hatch path) and
// the options needed to build the corresponding wrapper.
func resolveContextType(def *contextDefinition) (*runner.Binary, []runner.ExecutionContextOption, error) {
	switch def.Type {
	case "", "local":
		if err := checkForeignBlocks(def, def.Type); err != nil {
			return nil, nil, err
		}

		return &def.Executable, nil, nil
	case "docker":
		if err := checkForeignBlocks(def, "docker"); err != nil {
			return nil, nil, err
		}
		if def.Docker == nil {
			return nil, nil, fmt.Errorf("context type %q requires a docker block", def.Type)
		}
		if err := checkNoExecutable(def); err != nil {
			return nil, nil, err
		}
		if (def.Docker.Image == "") == (def.Docker.Container == "") {
			return nil, nil, fmt.Errorf("docker context requires exactly one of image or container")
		}

		spec := runner.DockerSpec{
			Image:     def.Docker.Image,
			Container: def.Docker.Container,
			Host:      def.Docker.Host,
			Options:   def.Docker.Options,
			Shell:     def.Docker.Shell,
		}

		return nil, []runner.ExecutionContextOption{runner.WithDocker(spec)}, nil
	case "kubernetes":
		if err := checkForeignBlocks(def, "kubernetes"); err != nil {
			return nil, nil, err
		}
		if def.Kubernetes == nil || def.Kubernetes.Pod == "" {
			return nil, nil, fmt.Errorf("kubernetes context requires pod")
		}
		if err := checkNoExecutable(def); err != nil {
			return nil, nil, err
		}

		spec := runner.KubernetesSpec{
			Pod:         def.Kubernetes.Pod,
			Namespace:   def.Kubernetes.Namespace,
			Container:   def.Kubernetes.Container,
			KubeContext: def.Kubernetes.KubeContext,
			Kubeconfig:  def.Kubernetes.Kubeconfig,
			Options:     def.Kubernetes.Options,
			Shell:       def.Kubernetes.Shell,
		}

		return nil, []runner.ExecutionContextOption{runner.WithKubernetes(spec)}, nil
	case "ssh":
		if err := checkForeignBlocks(def, "ssh"); err != nil {
			return nil, nil, err
		}
		if def.SSH == nil || def.SSH.Host == "" {
			return nil, nil, fmt.Errorf("ssh context requires host")
		}
		if err := checkNoExecutable(def); err != nil {
			return nil, nil, err
		}

		spec := runner.SSHSpec{
			Host:         def.SSH.Host,
			User:         def.SSH.User,
			Port:         def.SSH.Port,
			IdentityFile: def.SSH.IdentityFile,
			Options:      def.SSH.Options,
		}

		return nil, []runner.ExecutionContextOption{runner.WithSSH(spec)}, nil
	default:
		return nil, nil, fmt.Errorf("unknown context type %q", def.Type)
	}
}

// checkForeignBlocks rejects type-specific blocks that don't belong to typ, e.g.
// a "docker"-typed context that also sets a ssh: block.
func checkForeignBlocks(def *contextDefinition, typ string) error {
	if typ != "docker" && def.Docker != nil {
		return fmt.Errorf("context type %q does not accept a docker block", typ)
	}
	if typ != "kubernetes" && def.Kubernetes != nil {
		return fmt.Errorf("context type %q does not accept a kubernetes block", typ)
	}
	if typ != "ssh" && def.SSH != nil {
		return fmt.Errorf("context type %q does not accept a ssh block", typ)
	}

	return nil
}

// checkNoExecutable rejects the escape-hatch executable on typed contexts, since
// the wrapper drives command execution instead.
func checkNoExecutable(def *contextDefinition) error {
	if def.Executable.Bin != "" || len(def.Executable.Args) > 0 {
		return fmt.Errorf("context type %q does not accept an executable block", def.Type)
	}

	return nil
}
