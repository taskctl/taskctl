package executor

import (
	"github.com/taskctl/taskctl/pkg/variables"
	"io"
	"time"
)

// Job is a linked list of jobs to execute by Executor
type Job struct {
	Command string
	Dir     string
	Env     variables.Container
	Vars    variables.Container
	Timeout *time.Duration

	Stdout, Stderr io.Writer
	Stdin          io.Reader

	Next *Job
}

func NewJobFromCommand(command string) *Job {
	return &Job{
		Command: command,
		Vars:    variables.NewVariables(),
		Env:     variables.NewVariables(),
	}
}
