package config

import (
	"time"

	"github.com/Ensono/taskctl/pkg/utils"
)

//go:generate go run ../../tools/schemagenerator/main.go -dir ../../

// ConfigDefinition holds the top most config definition
// this can be parsed from a yaml, json, toml files/mediaTypes
//
// Note: works the same way for local/remote config resources
// correct content-type or file extension must be specified
// to successfully decode/unmarshal into type
type ConfigDefinition struct {
	// Import is a list of additional resources to bring into the main config
	// these can be remote or local resources
	Import []string `mapstructure:"import" yaml:"import" json:"import,omitempty"`
	// Contexts is a map of contexts to use
	// for specific tasks
	Contexts map[string]*ContextDefinition `mapstructure:"contexts" yaml:"contexts" json:"contexts,omitempty"`
	// Pipelines are a set of tasks wrapped in additional run conditions
	// e.g. depends on or allow failure
	Pipelines map[string][]*PipelineDefinition `mapstructure:"pipelines" yaml:"pipelines" json:"pipelines,omitempty"`
	// Tasks are the most basic buidling blocks
	Tasks    map[string]*TaskDefinition    `mapstructure:"tasks" yaml:"tasks" json:"tasks"`
	Watchers map[string]*WatcherDefinition `mapstructure:"watchers" yaml:"watchers" json:"watchers,omitempty"`
	Debug    bool                          `json:"debug,omitempty"`
	DryRun   bool                          `json:"dry_run,omitempty"`
	Summary  bool                          `json:"summary,omitempty"`
	// Output sets globally the output type for all tasks and pipelines
	//
	Output string `mapstructure:"output" yaml:"output" json:"output,omitempty" jsonschema:"enum=raw,enum=cockpit,enum=prefixed"`
	// Variables are the top most variables and will be merged and overwritten with lower level
	// specifications.
	// e.g. variable of Version=123
	// will be overwritten by a variables specified in this order, lowest takes the highest precedence
	// - context
	// - pipeline
	// - task
	// - commandline.
	// Variables can be used inside templating using the text/template go package
	Variables map[string]string `mapstructure:"variables" yaml:"variables" json:"variables,omitempty"` // jsonschema:"additional_properties_type=string;integer"`
}

type ContextDefinition struct {
	Dir    string   `mapstructure:"dir" yaml:"dir" json:"dir,omitempty"`
	Up     []string `mapstructure:"up" yaml:"up" json:"up,omitempty"`
	Down   []string `mapstructure:"down" yaml:"down" json:"down,omitempty"`
	Before []string `mapstructure:"before" yaml:"before" json:"before,omitempty"`
	After  []string `mapstructure:"after" yaml:"after" json:"after,omitempty"`
	// Env is supplied from config file definition and is merged with the
	// current process environemnt variables list
	//
	// User supplied env map will overwrite any keys inside the process env
	// TODO: check this is desired behaviour
	Env map[string]string `mapstructure:"env" yaml:"env" json:"env,omitempty"`
	// Envfile is a special block for use in executables that support file mapping
	// e.g. podman or docker
	//
	// the generated outputs will be merged with existing os.Environ()
	Envfile *utils.Envfile `mapstructure:"envfile" yaml:"envfile,omitempty" json:"envfile,omitempty"`
	// Variables
	Variables map[string]string `mapstructure:"variables" yaml:"variables" json:"variables,omitempty"`
	// Executable block holds the exec info
	Executable *utils.Binary `mapstructure:"executable" yaml:"executable" json:"executable,omitempty"`
	// Quote is the quote char to use when parsing commands into non-mvdan shells
	Quote string `mapstructure:"quote" yaml:"quote" json:"quote,omitempty"`
}

type PipelineDefinition struct {
	// Name is the friendly name to give to pipeline
	Name string `mapstructure:"name" yaml:"name" json:"name,omitempty"`
	// Condition is the condition to evaluate on whether to run this task or not
	Condition string `mapstructure:"condition" yaml:"condition" json:"condition,omitempty"`
	// Task is the pointer to the task to run
	// it has to match the key in tasks map
	Task string `mapstructure:"task" yaml:"task,omitempty" json:"task,omitempty" jsonschema:"oneof_required=task"`
	// Pipeline is the name of the pipeline to run
	// Task and Pipeline are mutually exclusive
	// if both are specified task will win
	Pipeline string `mapstructure:"pipeline" yaml:"pipeline,omitempty" json:"pipeline,omitempty" jsonschema:"oneof_required=pipeline"`
	// DependsOn
	DependsOn []string `mapstructure:"depends_on" yaml:"depends_on,omitempty" json:"depends_on,omitempty" jsonschema:"oneof_type=string;array"`
	// AllowFailure
	AllowFailure bool `mapstructure:"allow_failure" yaml:"allow_failure,omitempty" json:"allow_failure,omitempty"`
	// Dir is the place where to run the task(s) in.
	// If empty - currentDir is used
	Dir string `mapstructure:"dir" yaml:"dir,omitempty" json:"dir,omitempty"`
	// Env is the Key: Value map of env vars to inject into the tasks
	Env map[string]string `mapstructure:"env" yaml:"env,omitempty" json:"env,omitempty"`
	// Variables is the Key: Value map of vars vars to inject into the tasks
	Variables map[string]string `mapstructure:"variables" yaml:"variables,omitempty" json:"variables,omitempty"`
}

type TaskDefinition struct {
	Name        string `mapstructure:"name" yaml:"name,omitempty" json:"name,omitempty"`
	Description string `mapstructure:"description" yaml:"description,omitempty" json:"description,omitempty"`
	Condition   string `mapstructure:"condition" yaml:"condition,omitempty" json:"condition,omitempty"`
	// Command is the actual command to run in either a specified executable or
	// in mvdn.shell
	Command []string `mapstructure:"command" yaml:"command" json:"command" jsonschema:"oneof_type=string;array"`
	After   []string `mapstructure:"after" yaml:"after,omitempty" json:"after,omitempty"`
	Before  []string `mapstructure:"before" yaml:"before,omitempty" json:"before,omitempty"`
	// Context is the pointer to the key in the context map
	// it must exist else it will fallback to default context
	Context string `mapstructure:"context" yaml:"context,omitempty" json:"context,omitempty"`
	// Variations is per execution env var mutator
	// the number of variations in the list defines the number of times the command will be run
	// if using the default executor, see `ResetContext` if you need
	Variations []map[string]string `mapstructure:"variations" yaml:"variations,omitempty" json:"variations,omitempty"`
	// Dir to run the command from
	// If empty defaults to current directory
	Dir          string            `mapstructure:"dir" yaml:"dir,omitempty" json:"dir,omitempty"`
	Timeout      *time.Duration    `mapstructure:"timeout" yaml:"timeout,omitempty" json:"timeout,omitempty"`
	AllowFailure bool              `mapstructure:"allow_failure" yaml:"allow_failure,omitempty" json:"allow_failure,omitempty"`
	Interactive  bool              `mapstructure:"interactive" yaml:"interactive,omitempty" json:"interactive,omitempty"`
	ExportAs     string            `mapstructure:"export_as" yaml:"export_as,omitempty" json:"export_as,omitempty"`
	Env          map[string]string `mapstructure:"env" yaml:"env,omitempty" json:"env,omitempty"`
	// EnvFile string pointing to the file that could be read in as an envFile
	// contents will be merged with the Env (os.Environ())
	EnvFile string `mapstructure:"env_file" yaml:"env_file,omitempty" json:"env_file,omitempty"`
	// Variables merged with others if any already priovided
	// These will overwrite any previously set keys
	Variables map[string]string `mapstructure:"variables" yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"oneof_type=string;integer"`
	// ResetContext ensures each invocation of the variation is runs a Reset on the executor.
	// Currently only applies to a default executor.
	ResetContext bool `mapstructure:"reset_context" yaml:"reset_context,omitempty" json:"reset_context,omitempty" jsonschema:"default=false"`
}

type WatcherDefinition struct {
	Events    []string          `mapstructure:"events" yaml:"events" json:"events"`
	Watch     []string          `mapstructure:"watch" yaml:"watch" json:"watch"`
	Exclude   []string          `mapstructure:"exclude" yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Task      string            `mapstructure:"task" yaml:"task" json:"task"`
	Variables map[string]string `mapstructure:"variables" yaml:"variables,omitempty" json:"variables,omitempty"`
}
