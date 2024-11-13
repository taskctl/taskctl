package schema

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"
)

// Workflow represents the root of a GitHub workflow file.
type GithubWorkflow struct {
	Name        string               `json:"name,omitempty" yaml:"name,omitempty"`
	On          *GithubTriggerEvents `json:"on" yaml:"on"`
	Jobs        yaml.MapSlice        `json:"jobs" yaml:"jobs"`
	Defaults    *GithubDefaults      `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Env         map[string]any       `json:"env,omitempty" yaml:"env,omitempty"`
	Permissions map[string]string    `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

// TriggerEvents represents the trigger events for the GitHub workflow.
type GithubTriggerEvents struct {
	Push             GithubPushEvent             `json:"push,omitempty" yaml:"push,omitempty"`
	PullRequest      GithubPullRequestEvent      `json:"pull_request,omitempty" yaml:"pull_request,omitempty"`
	Schedule         []GithubScheduleEvent       `json:"schedule,omitempty" yaml:"schedule,omitempty"`
	WorkflowDispatch GithubWorkflowDispatchEvent `json:"workflow_dispatch,omitempty" yaml:"workflow_dispatch,omitempty"`
	// Other events can be added here as needed
}

// PushEvent represents a push event trigger configuration.
type GithubPushEvent struct {
	Branches       []string `json:"branches,omitempty" yaml:"branches,omitempty"`
	Tags           []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Paths          []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	PathsIgnore    []string `json:"paths-ignore,omitempty" yaml:"paths_ignore,omitempty"`
	BranchesIgnore []string `json:"branches-ignore,omitempty" yaml:"branches_ignore,omitempty"`
	TagsIgnore     []string `json:"tags-ignore,omitempty" yaml:"tags_ignore,omitempty"`
}

// PullRequestEvent represents a pull request event trigger configuration.
type GithubPullRequestEvent struct {
	Branches       []string `json:"branches,omitempty" yaml:"branches,omitempty"`
	Paths          []string `json:"paths,omitempty" yaml:"paths,omitempty"`
	BranchesIgnore []string `json:"branches-ignore,omitempty" yaml:"branches_ignore,omitempty"`
	PathsIgnore    []string `json:"paths-ignore,omitempty" yaml:"paths_ignore,omitempty"`
	Types          []string `json:"types,omitempty" yaml:"types,omitempty"`
}

// ScheduleEvent represents a cron schedule event trigger configuration.
type GithubScheduleEvent struct {
	Cron string `json:"cron,omitempty" yaml:"cron,omitempty"`
}

// WorkflowDispatchEvent represents a manually triggered workflow dispatch event.
type GithubWorkflowDispatchEvent struct {
	Inputs map[string]GithubInput `json:"inputs,omitempty" yaml:"inputs,omitempty"`
}

// Input represents an input for a workflow dispatch event.
type GithubInput struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
}

// Job represents a job in the GitHub workflow.
type GithubJob struct {
	Name           string           `json:"name,omitempty" yaml:"name,omitempty"`
	Needs          []string         `json:"needs,omitempty" yaml:"needs,omitempty"`
	RunsOn         string           `json:"runs-on,omitempty" yaml:"runs-on,omitempty"`
	Container      *GithubContainer `json:"container,omitempty" yaml:"container,omitempty"`
	Steps          []*GithubStep    `json:"steps,omitempty" yaml:"steps,omitempty"`
	If             string           `json:"if,omitempty" yaml:"if,omitempty"`
	Env            map[string]any   `json:"env,omitempty" yaml:"env,omitempty"`
	Environment    string           `json:"environment,omitempty" yaml:"environment,omitempty"`
	TimeoutMinutes int              `json:"timeout-minutes,omitempty" yaml:"timeout_minutes,omitempty"`
	Strategy       *GithubStrategy  `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

var ErrMustIncludeSubComponents = errors.New("must include at least one")

func (job *GithubJob) AddStep(step *GithubStep) error {
	if step != nil {
		job.Steps = append(job.Steps, step)
		return nil
	}
	return fmt.Errorf("github job, %w", ErrMustIncludeSubComponents)
}

// Container represents a container configuration for a GitHub Action job.
type GithubContainer struct {
	Image       string                      `json:"image" yaml:"image"`                                 // The Docker image to use for the container.
	Credentials *GithubContainerCredentials `json:"credentials,omitempty" yaml:"credentials,omitempty"` // Credentials for the container registry, if required.
	Env         map[string]string           `json:"env,omitempty" yaml:"env,omitempty"`                 // Environment variables for the container.
	Ports       []interface{}               `json:"ports,omitempty" yaml:"ports,omitempty"`             // Array of ports to expose, can be numbers or strings.
	Volumes     []string                    `json:"volumes,omitempty" yaml:"volumes,omitempty"`         // Array of volumes to use in the container.
	Options     string                      `json:"options,omitempty" yaml:"options,omitempty"`         // Additional Docker container options.
}

// ContainerCredentials represents credentials for the container registry.
type GithubContainerCredentials struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"` // Username for the container registry.
	Password string `json:"password,omitempty" yaml:"password,omitempty"` // Password for the container registry.
}

// Step represents a step in a GitHub job.
type GithubStep struct {
	Name            string            `json:"name,omitempty" yaml:"name,omitempty"`
	ID              string            `json:"id,omitempty" yaml:"id,omitempty"`
	Uses            string            `json:"uses,omitempty" yaml:"uses,omitempty"`
	Run             string            `json:"run,omitempty" yaml:"run,omitempty"`
	With            map[string]string `json:"with,omitempty" yaml:"with,omitempty"`
	Env             map[string]any    `json:"env,omitempty" yaml:"env,omitempty"`
	Shell           string            `json:"shell,omitempty" yaml:"shell,omitempty"`
	ContinueOnError bool              `json:"continue-on-error,omitempty" yaml:"continue_on_error,omitempty"`
	TimeoutMinutes  int               `json:"timeout-minutes,omitempty" yaml:"timeout_minutes,omitempty"`
	If              string            `json:"if,omitempty" yaml:"if,omitempty"`
}

// Defaults represents default values for jobs in the GitHub workflow.
type GithubDefaults struct {
	Run GithubDefaultRun `json:"run,omitempty" yaml:"run,omitempty"`
}

// DefaultRun represents default run configurations for the jobs in the workflow.
type GithubDefaultRun struct {
	Shell            string `json:"shell,omitempty" yaml:"shell,omitempty"`
	WorkingDirectory string `json:"working-directory,omitempty" yaml:"working_directory,omitempty"`
}

// Strategy represents a job strategy (matrix) configuration.
type GithubStrategy struct {
	Matrix      map[string][]string `json:"matrix,omitempty" yaml:"matrix,omitempty"`
	MaxParallel int                 `json:"max-parallel,omitempty" yaml:"max_parallel,omitempty"`
	FailFast    bool                `json:"fail-fast,omitempty" yaml:"fail_fast,omitempty"`
}
