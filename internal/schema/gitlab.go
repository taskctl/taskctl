// Package gitlab was largely generated by an AI from the schema
// There doesn't seem to be a reliable way of generating types (go structs) from the CI schema
//
// tried quicktype and
package schema

// GitLabCI represents the root structure of a GitLab CI pipeline file.
type GitLabCI struct {
	Version      string               `json:"version,omitempty" yaml:"version,omitempty"`
	Stages       []string             `json:"stages,omitempty" yaml:"stages,omitempty"`
	Variables    map[string]string    `json:"variables,omitempty" yaml:"variables,omitempty"`
	Include      []GitLabInclude      `json:"include,omitempty" yaml:"include,omitempty"`
	Jobs         map[string]GitLabJob `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	BeforeScript []string             `json:"before_script,omitempty" yaml:"before_script,omitempty"`
	AfterScript  []string             `json:"after_script,omitempty" yaml:"after_script,omitempty"`
	Image        GitLabImage          `json:"image,omitempty" yaml:"image,omitempty"`
	Services     []GitLabService      `json:"services,omitempty" yaml:"services,omitempty"`
}

// Include represents external files that can be included into the CI configuration.
type GitLabInclude struct {
	Local    string `json:"local,omitempty" yaml:"local,omitempty"`
	File     string `json:"file,omitempty" yaml:"file,omitempty"`
	Template string `json:"template,omitempty" yaml:"template,omitempty"`
	Remote   string `json:"remote,omitempty" yaml:"remote,omitempty"`
	Ref      string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Project  string `json:"project,omitempty" yaml:"project,omitempty"`
}

// Job represents a single job definition in GitLab CI.
type GitLabJob struct {
	Script       []string           `json:"script,omitempty" yaml:"script,omitempty"`
	Stage        string             `json:"stage,omitempty" yaml:"stage,omitempty"`
	Tags         []string           `json:"tags,omitempty" yaml:"tags,omitempty"`
	Only         GitLabJobCondition `json:"only,omitempty" yaml:"only,omitempty"`
	Except       GitLabJobCondition `json:"except,omitempty" yaml:"except,omitempty"`
	Variables    map[string]string  `json:"variables,omitempty" yaml:"variables,omitempty"`
	When         string             `json:"when,omitempty" yaml:"when,omitempty"`
	AllowFailure bool               `json:"allow_failure,omitempty" yaml:"allow_failure,omitempty"`
	BeforeScript []string           `json:"before_script,omitempty" yaml:"before_script,omitempty"`
	AfterScript  []string           `json:"after_script,omitempty" yaml:"after_script,omitempty"`
	Dependencies []string           `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Artifacts    GitLabArtifacts    `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Retry        GitLabRetry        `json:"retry,omitempty" yaml:"retry,omitempty"`
}

// Image represents the Docker image configuration.
type GitLabImage struct {
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	EntryPoint string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
}

// Service represents services that are used during the job execution.
type GitLabService struct {
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	Alias      string `json:"alias,omitempty" yaml:"alias,omitempty"`
	EntryPoint string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Command    string `json:"command,omitempty" yaml:"command,omitempty"`
}

// JobCondition represents conditions for when a job runs.
type GitLabJobCondition struct {
	Refs       []string `json:"refs,omitempty" yaml:"refs,omitempty"`
	Variables  []string `json:"variables,omitempty" yaml:"variables,omitempty"`
	Kubernetes []string `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
}

// Artifacts represents the configuration for job artifacts.
type GitLabArtifacts struct {
	Paths    []string      `json:"paths,omitempty" yaml:"paths,omitempty"`
	Exclude  []string      `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	ExpireIn string        `json:"expire_in,omitempty" yaml:"expire_in,omitempty"`
	When     string        `json:"when,omitempty" yaml:"when,omitempty"`
	Reports  GitLabReports `json:"reports,omitempty" yaml:"reports,omitempty"`
}

// Reports represent specific test reports that are uploaded after a job.
type GitLabReports struct {
	JUnit     string   `json:"junit,omitempty" yaml:"junit,omitempty"`
	Artifacts []string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
}

// Retry represents the retry configuration for a job.
type GitLabRetry struct {
	Max  int      `json:"max,omitempty" yaml:"max,omitempty"`
	When []string `json:"when,omitempty" yaml:"when,omitempty"`
}