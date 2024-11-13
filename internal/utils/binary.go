package utils

import (
	"slices"
	"strings"
)

// Binary is a structure for storing binary file path and arguments that should be passed on binary's invocation
type Binary struct {
	// IsContainer marks the binary as the container native implementation
	IsContainer bool `jsonschema:"-"`
	// Bin is the name of the executable to run
	// it must exist on the path
	// If using a default mvdn.sh context then
	// ensure it is on your path as symlink if you are only using aliases.
	Bin  string   `mapstructure:"bin" yaml:"bin" json:"bin"`
	Args []string `mapstructure:"args" yaml:"args,omitempty" json:"args,omitempty"`

	// baseArgs will be the default args always managed by taskctl
	// the last item will always be --envfile
	baseArgs      []string
	containerArgs []string
	shellArgs     []string
}

func (b *Binary) WithBaseArgs(args []string) *Binary {
	b.baseArgs = args
	return b
}

func (b *Binary) WithShellArgs(args []string) *Binary {
	b.shellArgs = args
	return b
}

func (b *Binary) WithContainerArgs(args []string) *Binary {
	b.containerArgs = args
	return b
}

func (b *Binary) GetArgs() []string {
	return b.Args
}

func (b *Binary) buildContainerArgsWithEnvFile(envfilePath string) []string {
	outArgs := append(b.baseArgs, envfilePath)
	outArgs = append(outArgs, b.containerArgs...)
	outArgs = append(outArgs, b.shellArgs...)
	return outArgs
}

// BuildArgsWithEnvFile returns all args with
// correctly placed --env-file parameter for
// context binary
func (b *Binary) BuildArgsWithEnvFile(envfilePath string) []string {
	if b.IsContainer {
		return b.buildContainerArgsWithEnvFile(envfilePath)
	}

	// slices are basically pointers so we want to copy the data
	outArgs := make([]string, len(b.Args))
	copy(outArgs, b.Args)

	// sanitize the bin params this is a legacy method
	if slices.Contains([]string{"docker", "podman"}, strings.ToLower(b.Bin)) {
		// does the args contain the --env-file string
		// currently we will always either overwrite or just append the `--env-file flag`
		idx := slices.Index(outArgs, "--env-file")
		// the envfile has been added to the args, need to overwrite the value
		if idx > -1 {
			outArgs[idx+1] = envfilePath
			return outArgs
		}

		// the envfile has NOT been added to the args, so this needs to be added in
		// as the docker args order is important, these will be prepended to the array
		outArgs = append([]string{outArgs[0], "--env-file", envfilePath}, outArgs[1:]...)
		return outArgs
	}
	return outArgs
}

// Container is the specific context for containers
// only available to docker API compliant implementations
//
// e.g. docker and podman
//
// The aim is to remove some of the boilerplate away from the existing more
// generic context and introduce a specific context for tasks run in containers.
type Container struct {
	// Name is the name of the container
	//
	// can be specified in the following formats
	//
	// - <image-name> (Same as using <image-name> with the latest tag)
	//
	// - <image-name>:<tag>
	//
	// - <image-name>@<digest>
	//
	// If the known runtime is podman it should include the registry domain
	// e.g. `docker.io/alpine:latest`
	Name string `mapstructure:"name" yaml:"name" json:"name"`
	// Entrypoint Overwrites the default ENTRYPOINT of the image
	Entrypoint string `mapstructure:"entrypoint" yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	// EnableDinD mounts the docker sock...
	//
	// >highly discouraged
	EnableDinD bool `mapstructure:"enable_dind" yaml:"enable_dind,omitempty" json:"enable_dind,omitempty"`
	// ContainerArgs are additional args used for the container supplied by the user
	//
	// e.g. dcoker run (TASKCTL_ARGS...) (CONTAINER_ARGS...) image (command)
	// The internals will strip out any unwanted/forbidden args
	//
	// Args like the switch --privileged and the --volume|-v flag with the value of /var/run/docker.sock:/var/run/docker.sock
	// will be removed.
	ContainerArgs []string `mapstructure:"container_args" yaml:"container_args,omitempty" json:"container_args,omitempty"`
	// Shell will be used to run the command in a specific shell on the container
	//
	// Must exist in the container
	Shell string `mapstructure:"shell" yaml:"shell,omitempty" json:"shell,omitempty"`
	// Args are additional args to pass to the shell if provided
	//
	// // e.g. dcoker run (TASKCTL_ARGS...) (CONTAINER_ARGS...) image (shell) (SHELL_ARGS...) (command)
	ShellArgs []string `mapstructure:"shell_args" yaml:"shell_args,omitempty" json:"shell_args,omitempty"`
}
