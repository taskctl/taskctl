package runner

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// DockerSpec configures a wrapper that runs commands inside a Docker container.
type DockerSpec struct {
	// Image selects run mode (a fresh container per command); Container selects exec mode. Exactly one must be set.
	Image     string
	Container string
	Host      string
	Options   []string
	Shell     []string
}

// KubernetesSpec configures a wrapper that runs commands via kubectl exec.
type KubernetesSpec struct {
	Pod         string
	Namespace   string
	Container   string
	KubeContext string
	Kubeconfig  string
	Options     []string
	Shell       []string
}

// SSHSpec configures a wrapper that runs commands over ssh.
type SSHSpec struct {
	Host         string
	User         string
	Port         int
	IdentityFile string
	Options      []string
	Shell        []string
}

// commandWrapper builds the local shell command that executes command inside a target (container, pod, or remote host) with env/dir applied there.
type commandWrapper interface {
	wrap(command string, env map[string]string, dir string) string
}

type dockerWrapper struct {
	spec DockerSpec
}

type kubectlWrapper struct {
	spec KubernetesSpec
}

type sshWrapper struct {
	spec SSHSpec
}

func newDockerWrapper(spec DockerSpec) commandWrapper {
	if len(spec.Shell) == 0 {
		spec.Shell = defaultShell()
	}
	return &dockerWrapper{spec: spec}
}

func newKubectlWrapper(spec KubernetesSpec) commandWrapper {
	if len(spec.Shell) == 0 {
		spec.Shell = defaultShell()
	}
	return &kubectlWrapper{spec: spec}
}

func newSSHWrapper(spec SSHSpec) commandWrapper {
	if len(spec.Shell) == 0 {
		spec.Shell = defaultShell()
	}
	return &sshWrapper{spec: spec}
}

func (w *dockerWrapper) wrap(command string, env map[string]string, dir string) string {
	tokens := []string{"docker"}
	if w.spec.Host != "" {
		tokens = append(tokens, "-H", shellQuote(w.spec.Host))
	}

	if w.spec.Image != "" {
		tokens = append(tokens, "run", "--rm")
	} else {
		tokens = append(tokens, "exec")
	}

	tokens = append(tokens, envFlags(env)...)

	if dir != "" {
		tokens = append(tokens, "-w", shellQuote(dir))
	}

	tokens = append(tokens, w.spec.Options...)

	if w.spec.Image != "" {
		tokens = append(tokens, shellQuote(w.spec.Image))
	} else {
		tokens = append(tokens, shellQuote(w.spec.Container))
	}

	tokens = append(tokens, w.spec.Shell...)
	tokens = append(tokens, shellQuote(command))

	return strings.Join(tokens, " ")
}

func (w *kubectlWrapper) wrap(command string, env map[string]string, dir string) string {
	tokens := []string{"kubectl"}
	if w.spec.Kubeconfig != "" {
		tokens = append(tokens, "--kubeconfig", shellQuote(w.spec.Kubeconfig))
	}
	if w.spec.KubeContext != "" {
		tokens = append(tokens, "--context", shellQuote(w.spec.KubeContext))
	}
	if w.spec.Namespace != "" {
		tokens = append(tokens, "-n", shellQuote(w.spec.Namespace))
	}

	tokens = append(tokens, "exec")
	tokens = append(tokens, w.spec.Options...)
	tokens = append(tokens, shellQuote(w.spec.Pod))

	if w.spec.Container != "" {
		tokens = append(tokens, "-c", shellQuote(w.spec.Container))
	}

	tokens = append(tokens, "--")
	tokens = append(tokens, w.spec.Shell...)
	tokens = append(tokens, shellQuote(buildScript(env, dir, command)))

	return strings.Join(tokens, " ")
}

func (w *sshWrapper) wrap(command string, env map[string]string, dir string) string {
	tokens := []string{"ssh"}
	if w.spec.Port != 0 {
		tokens = append(tokens, "-p", shellQuote(strconv.Itoa(w.spec.Port)))
	}
	if w.spec.IdentityFile != "" {
		tokens = append(tokens, "-i", shellQuote(w.spec.IdentityFile))
	}

	tokens = append(tokens, w.spec.Options...)

	target := w.spec.Host
	if w.spec.User != "" {
		target = w.spec.User + "@" + w.spec.Host
	}
	tokens = append(tokens, shellQuote(target))
	tokens = append(tokens, shellQuote(buildScript(env, dir, command)))

	return strings.Join(tokens, " ")
}

// shellQuote POSIX-quotes s for the local shell parse; it is the single choke point for local-parse quoting.
func shellQuote(s string) string {
	quoted, err := syntax.Quote(s, syntax.LangPOSIX)
	if err != nil {
		return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
	}
	return quoted
}

// buildScript builds the in-target script that exports env, cds into dir, then runs command.
func buildScript(env map[string]string, dir, command string) string {
	var b strings.Builder

	if len(env) > 0 {
		b.WriteString("export ")
		keys := slices.Sorted(maps.Keys(env))
		for i, k := range keys {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(k)
			b.WriteString("=")
			b.WriteString(shellQuote(env[k]))
		}
		b.WriteString("; ")
	}

	if dir != "" {
		b.WriteString("cd ")
		b.WriteString(shellQuote(dir))
		b.WriteString(" && ")
	}

	b.WriteString(command)

	return b.String()
}

// envFlags returns docker -e flags for env, one pair of tokens per sorted key.
func envFlags(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	keys := slices.Sorted(maps.Keys(env))
	flags := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		flags = append(flags, "-e", shellQuote(k+"="+env[k]))
	}
	return flags
}

func defaultShell() []string {
	return []string{"sh", "-c"}
}
